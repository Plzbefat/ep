//用于服务器本地开发时数据库代理
package ep

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/ioutil"
	"net"

	"github.com/go-redis/redis"
	sqlDriver "github.com/go-sql-driver/mysql"
	"golang.org/x/crypto/ssh"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

type Config struct {
	remoteServerAddress string
	localKeyFilePath    string
	isDebug             bool
}

//设置代理配置
func NewProxy(remoteServerAddress, localKeyFilePath string, isDebug bool) *Config {
	return &Config{
		remoteServerAddress: remoteServerAddress,
		localKeyFilePath:    localKeyFilePath,
		isDebug:             isDebug,
	}
}

//生成ssh远程代理
func (c *Config) getSshProxyClient() (*ssh.Client, error) {
	if c.remoteServerAddress == "" {
		return nil, errors.New("remoteServerAddress is empty")
	}

	if c.localKeyFilePath == "" {
		return nil, errors.New("localKeyFilePath is empty")
	}

	key, readFileErr := ioutil.ReadFile(c.localKeyFilePath)
	if readFileErr != nil {
		return nil, readFileErr
	}

	signer, parsePrivateKeyErr := ssh.ParsePrivateKey(key)
	if parsePrivateKeyErr != nil {
		return nil, parsePrivateKeyErr
	}

	return ssh.Dial("tcp", c.remoteServerAddress, &ssh.ClientConfig{
		User:            "root",
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	})
}

//debug?代理:本地 redis
func (c *Config) GetRedisClient(link, password string, db int) (*redis.Client, error) {
	if c.isDebug {
		return c.getProxyRedisClient(link, password, db)
	} else {
		return getRedisClient(link, password, db), nil
	}
}

//debug?代理:本地 mysql
func (c *Config) GetMysqlClient(link, password, db string) (*gorm.DB, error) {
	if c.isDebug {
		return c.getProxyMysqlClient(link, password, db)
	} else {
		return getMysqlClient(link, password, db), nil
	}
}

//通过代理连接到redis
func (c *Config) getProxyRedisClient(link, password string, db int) (*redis.Client, error) {
	sshProxyConn, err := c.getSshProxyClient()
	if err != nil {
		return nil, err
	}

	return redis.NewClient(&redis.Options{
		Dialer: func() (conn net.Conn, e error) {
			return sshProxyConn.Dial("tcp", link)
		},
		Password: password,
		DB:       db,
	}), nil
}

//通过代理连接到mysql
func (c *Config) getProxyMysqlClient(link, password, db string) (*gorm.DB, error) {
	sshProxyConn, err := c.getSshProxyClient()
	if err != nil {
		return nil, err
	}

	sqlDriver.RegisterDialContext("mysql_proxy_tcp", func(_ context.Context, addr string) (conn net.Conn, e error) {
		return sshProxyConn.Dial("tcp", link)
	})
	mysqlProxyConn, _err := sql.Open("mysql", fmt.Sprintf("root:%s@mysql_proxy_tcp(127.0.0.1)/%s?charset=utf8mb4&parseTime=True&loc=Local", password, db))
	if _err != nil {
		panic(_err)
	}

	return gorm.Open(mysql.New(mysql.Config{Conn: mysqlProxyConn}), &gorm.Config{NamingStrategy: schema.NamingStrategy{SingularTable: true}, Logger: logger.Default.LogMode(logger.Info)})
}

//直接连接到redis
func getRedisClient(link, password string, db int) *redis.Client {
	return redis.NewClient(&redis.Options{Addr: link, Password: password, DB: db})
}

//直接连接到mysql
func getMysqlClient(link, password, db string) *gorm.DB {
	mysqlClient, err := gorm.Open(mysql.Open(fmt.Sprintf("root:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", password, link, db)), &gorm.Config{NamingStrategy: schema.NamingStrategy{SingularTable: true}})
	if err != nil {
		panic(err)
	}
	return mysqlClient
}
