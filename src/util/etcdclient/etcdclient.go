package etcdclient

import (
	"context"
	"time"
	"fmt"
	"go.etcd.io/etcd/clientv3"
	"go.etcd.io/etcd/mvcc/mvccpb"
	"crypto/tls"
    "crypto/x509"
	"io/ioutil"
)

var Etcdclient *EtcdClient

type EtcdClient struct {
	Leaseid clientv3.LeaseID
	Client  *clientv3.Client
	Lease   clientv3.Lease
	DialTimeout	int
	RequestTimeout int
	Leasetime int64
	KeepResp *clientv3.LeaseKeepAliveResponse
	KeepRespChan <-chan *clientv3.LeaseKeepAliveResponse
}

// 常规初始化, 不带ca认证的情况下
func ClientInit(dialTimeout, requestTimeout int , leaseTime int64, endpoints []string) error {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: time.Duration(dialTimeout) * time.Second,
	})
	if err != nil {
		return fmt.Errorf("Init Common EtcdClient Error, ErrorInfo: %s", err.Error())
	}

	// 创建租约
	lease := clientv3.NewLease(cli)

	//设置租约时间
	leaseResp, err := lease.Grant(context.TODO(), leaseTime)
	if err != nil {
		return fmt.Errorf("Set Leasetime Error, ErrorInfo: %s", err.Error())
	}

	// 获取租约ID
	leaseId := leaseResp.ID

	// 创建一个自动续期的协程
	keepRespChan, err := lease.KeepAlive(context.TODO(), leaseId)
	if err != nil {
		return fmt.Errorf("Set Auto Lease Rerate Error, ErrorInfo: %s", err.Error())
	}	

	Etcdclient = &EtcdClient {
		Leaseid: leaseId,
		Client:  cli,
		Lease:  lease,
		DialTimeout: dialTimeout,
		RequestTimeout: requestTimeout,
		Leasetime: leaseTime,
		KeepRespChan: keepRespChan,
	}
	return nil 
}


// 安全初始化， 带ca认证的情况下
func ClientInitWitchCA(etcdCert, etcdCertKey, etcdCa string, dialTimeout, requestTimeout int, leaseTime int64, endpoints []string) error {
    cert, err := tls.LoadX509KeyPair(etcdCert, etcdCertKey)
    if err != nil {
        return fmt.Errorf("set Tls Cert Falied, ErrorInfo: %s", err.Error())
    }

    caData, err := ioutil.ReadFile(etcdCa)
    if err != nil {
        return fmt.Errorf("set caData Falied, ErrorInfo: %s", err.Error())
    }

    pool := x509.NewCertPool()
    pool.AppendCertsFromPEM(caData)

    _tlsConfig := &tls.Config{
        Certificates: []tls.Certificate{cert},
        RootCAs:      pool,
    }	

	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: time.Duration(dialTimeout) * time.Second,
		TLS:       _tlsConfig,
	})
	if err != nil {
		return fmt.Errorf("Init Common EtcdClient Error, ErrorInfo: %s", err.Error())
	}

	// 创建租约
	lease := clientv3.NewLease(cli)

	//设置租约时间
	leaseResp, err := lease.Grant(context.TODO(), leaseTime)
	if err != nil {
		return fmt.Errorf("Set Leasetime Error, ErrorInfo: %s", err.Error())
	}

	// 获取租约ID
	leaseId := leaseResp.ID

	// 创建一个自动续期的协程
	keepRespChan, err := lease.KeepAlive(context.TODO(), leaseId)
	if err != nil {
		return fmt.Errorf("Set Auto Lease Rerate Error, ErrorInfo: %s", err.Error())
	}	

	Etcdclient = &EtcdClient {
		Leaseid: leaseId,
		Client:  cli,
		Lease:  lease,
		DialTimeout: dialTimeout,
		RequestTimeout: requestTimeout,
		Leasetime: leaseTime,
		KeepRespChan: keepRespChan,
	}
	return nil
}


// 启动一个协程序用于后台自动续租
func (e *EtcdClient) LeaseReRate() {
	for {
		select {
		case e.KeepResp = <-e.KeepRespChan:
			if e.KeepRespChan == nil {
				fmt.Println("自动续期失败")
			} else { //每秒会续租一次，所以就会受到一次应答
				continue
			}
		}
	}
}

func (e *EtcdClient) Put(key, value string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(e.RequestTimeout) * time.Second)
	_, err := e.Client.Put(ctx, key, value, clientv3.WithLease(e.Leaseid))
	cancel()
	if err != nil {
		return fmt.Errorf("Put Data To Etcd Failed, Key: %s, Value: %s, Error Info: %s", key, value, err.Error())
	}
	return nil
}

func (e *EtcdClient) Get(key string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(e.RequestTimeout) * time.Second)
	getResp, err := e.Client.Get(ctx, key, clientv3.WithPrefix())
	cancel()
	if err != nil {
		return "", fmt.Errorf("Get Data From Etcd Failed, Key: %s, Error Info: %s", key, err.Error())
	}

	if len(getResp.Kvs) == 0 {
		return "", fmt.Errorf("Get Etcd Key Failed , Error Info: No Key")
	}

	return string(getResp.Kvs[0].Value), nil
}

func (e *EtcdClient) WatchCfg(key string) {
    ctx, cancel := context.WithCancel(context.Background())
	cancel()
    //这里ctx感知到cancel则会关闭watcher
    watchRespChan := e.Client.Watch(ctx, key)

    // 处理kv变化事件
    for watchResp := range watchRespChan {
        for _, event := range watchResp.Events {
            switch event.Type {
            case mvccpb.PUT:
                fmt.Println("Put:", string(event.Kv.Value))
            case mvccpb.DELETE:
                fmt.Println("Delete")
            }
        }
    }
}


