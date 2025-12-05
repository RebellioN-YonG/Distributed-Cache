package cache

import (
	// pb "cache/pb"
	"github.com/RebellioN-YonG/Distrbuted-Cache/store"
	clientv3 "go.etcd.io/etcd/client/v3"
	grpc "google.golang.org/grpc"
)

type Client struct {
	addr    string
	svcName string
	etcdCli *clientv3.Client
	conn    *grpc.ClientConn
	// grpcCli pb.CacheClient
	store   store.Store
}