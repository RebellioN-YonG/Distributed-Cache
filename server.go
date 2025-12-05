package rebelcache

import (
	"sync"

	"github.com/RebellioN-YonG/Distrbuted-Cache/store"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
)

type Server struct {
	addr       string           // server's addr
	svcName    string           // service name
	groups     *sync.Map        // cache groups
	grpcServer *grpc.Server     // grpc server
	etcdCli    *clientv3.Client // etcd client
	stopCh     chan error       // stop channel
	opts       *ServerOptions   // server options
	store      store.Store      // cache store
}

type ServerOptions struct {
	ServerAddr string
	EtcdAddr   string
}