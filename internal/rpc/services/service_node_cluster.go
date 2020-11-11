package services

import (
	"context"
	"encoding/json"
	"github.com/TeaOSLab/EdgeAPI/internal/db/models"
	"github.com/TeaOSLab/EdgeAPI/internal/dnsproviders"
	"github.com/TeaOSLab/EdgeAPI/internal/errors"
	rpcutils "github.com/TeaOSLab/EdgeAPI/internal/rpc/utils"
	"github.com/TeaOSLab/EdgeAPI/internal/tasks"
	"github.com/TeaOSLab/EdgeCommon/pkg/rpc/pb"
	"github.com/iwind/TeaGo/types"
	"strconv"
)

type NodeClusterService struct {
}

// 创建集群
func (this *NodeClusterService) CreateNodeCluster(ctx context.Context, req *pb.CreateNodeClusterRequest) (*pb.CreateNodeClusterResponse, error) {
	_, _, err := rpcutils.ValidateRequest(ctx, rpcutils.UserTypeAdmin)
	if err != nil {
		return nil, err
	}

	clusterId, err := models.SharedNodeClusterDAO.CreateCluster(req.Name, req.GrantId, req.InstallDir)
	if err != nil {
		return nil, err
	}

	return &pb.CreateNodeClusterResponse{ClusterId: clusterId}, nil
}

// 修改集群
func (this *NodeClusterService) UpdateNodeCluster(ctx context.Context, req *pb.UpdateNodeClusterRequest) (*pb.RPCUpdateSuccess, error) {
	_, _, err := rpcutils.ValidateRequest(ctx, rpcutils.UserTypeAdmin)
	if err != nil {
		return nil, err
	}

	err = models.SharedNodeClusterDAO.UpdateCluster(req.ClusterId, req.Name, req.GrantId, req.InstallDir)
	if err != nil {
		return nil, err
	}

	return &pb.RPCUpdateSuccess{}, nil
}

// 禁用集群
func (this *NodeClusterService) DeleteNodeCluster(ctx context.Context, req *pb.DeleteNodeClusterRequest) (*pb.RPCDeleteSuccess, error) {
	_, _, err := rpcutils.ValidateRequest(ctx, rpcutils.UserTypeAdmin)
	if err != nil {
		return nil, err
	}

	err = models.SharedNodeClusterDAO.DisableNodeCluster(req.ClusterId)
	if err != nil {
		return nil, err
	}

	return rpcutils.RPCDeleteSuccess()
}

// 查找单个集群
func (this *NodeClusterService) FindEnabledNodeCluster(ctx context.Context, req *pb.FindEnabledNodeClusterRequest) (*pb.FindEnabledNodeClusterResponse, error) {
	_, _, err := rpcutils.ValidateRequest(ctx, rpcutils.UserTypeAdmin)
	if err != nil {
		return nil, err
	}

	cluster, err := models.SharedNodeClusterDAO.FindEnabledNodeCluster(req.ClusterId)
	if err != nil {
		return nil, err
	}

	if cluster == nil {
		return &pb.FindEnabledNodeClusterResponse{}, nil
	}

	return &pb.FindEnabledNodeClusterResponse{Cluster: &pb.NodeCluster{
		Id:         int64(cluster.Id),
		Name:       cluster.Name,
		CreatedAt:  int64(cluster.CreatedAt),
		InstallDir: cluster.InstallDir,
		GrantId:    int64(cluster.GrantId),
		UniqueId:   cluster.UniqueId,
		Secret:     cluster.Secret,
	}}, nil
}

// 查找集群的API节点信息
func (this *NodeClusterService) FindAPINodesWithNodeCluster(ctx context.Context, req *pb.FindAPINodesWithNodeClusterRequest) (*pb.FindAPINodesWithNodeClusterResponse, error) {
	// 校验请求
	_, _, err := rpcutils.ValidateRequest(ctx, rpcutils.UserTypeAdmin)
	if err != nil {
		return nil, err
	}

	cluster, err := models.SharedNodeClusterDAO.FindEnabledNodeCluster(req.ClusterId)
	if err != nil {
		return nil, err
	}
	if cluster == nil {
		return nil, errors.New("can not find cluster with id '" + strconv.FormatInt(req.ClusterId, 10) + "'")
	}

	result := &pb.FindAPINodesWithNodeClusterResponse{}
	result.UseAllAPINodes = cluster.UseAllAPINodes == 1

	apiNodeIds := []int64{}
	if len(cluster.ApiNodes) > 0 && cluster.ApiNodes != "null" {
		err = json.Unmarshal([]byte(cluster.ApiNodes), &apiNodeIds)
		if err != nil {
			return nil, err
		}
		if len(apiNodeIds) > 0 {
			apiNodes := []*pb.APINode{}
			for _, apiNodeId := range apiNodeIds {
				apiNode, err := models.SharedAPINodeDAO.FindEnabledAPINode(apiNodeId)
				if err != nil {
					return nil, err
				}
				apiNodeAddrs, err := apiNode.DecodeAccessAddrStrings()
				if err != nil {
					return nil, err
				}
				apiNodes = append(apiNodes, &pb.APINode{
					Id:          int64(apiNode.Id),
					IsOn:        apiNode.IsOn == 1,
					ClusterId:   int64(apiNode.ClusterId),
					Name:        apiNode.Name,
					Description: apiNode.Description,
					AccessAddrs: apiNodeAddrs,
				})
			}
			result.ApiNodes = apiNodes
		}
	}

	return result, nil
}

// 查找所有可用的集群
func (this *NodeClusterService) FindAllEnabledNodeClusters(ctx context.Context, req *pb.FindAllEnabledNodeClustersRequest) (*pb.FindAllEnabledNodeClustersResponse, error) {
	// 校验请求
	_, _, err := rpcutils.ValidateRequest(ctx, rpcutils.UserTypeAdmin)
	if err != nil {
		return nil, err
	}

	clusters, err := models.SharedNodeClusterDAO.FindAllEnableClusters()
	if err != nil {
		return nil, err
	}

	result := []*pb.NodeCluster{}
	for _, cluster := range clusters {
		result = append(result, &pb.NodeCluster{
			Id:        int64(cluster.Id),
			Name:      cluster.Name,
			CreatedAt: int64(cluster.CreatedAt),
			UniqueId:  cluster.UniqueId,
			Secret:    cluster.Secret,
		})
	}

	return &pb.FindAllEnabledNodeClustersResponse{
		Clusters: result,
	}, nil
}

// 查找所有变更的集群
func (this *NodeClusterService) FindAllChangedNodeClusters(ctx context.Context, req *pb.FindAllChangedNodeClustersRequest) (*pb.FindAllChangedNodeClustersResponse, error) {
	_, _, err := rpcutils.ValidateRequest(ctx, rpcutils.UserTypeAdmin)
	if err != nil {
		return nil, err
	}

	clusterIds, err := models.SharedNodeDAO.FindChangedClusterIds()
	if err != nil {
		return nil, err
	}
	if len(clusterIds) == 0 {
		return &pb.FindAllChangedNodeClustersResponse{
			Clusters: []*pb.NodeCluster{},
		}, nil
	}
	result := []*pb.NodeCluster{}
	for _, clusterId := range clusterIds {
		cluster, err := models.SharedNodeClusterDAO.FindEnabledNodeCluster(clusterId)
		if err != nil {
			return nil, err
		}
		if cluster == nil {
			continue
		}
		result = append(result, &pb.NodeCluster{
			Id:        int64(cluster.Id),
			Name:      cluster.Name,
			CreatedAt: int64(cluster.CreatedAt),
			UniqueId:  cluster.UniqueId,
			Secret:    cluster.Secret,
		})
	}
	return &pb.FindAllChangedNodeClustersResponse{Clusters: result}, nil
}

// 计算所有集群数量
func (this *NodeClusterService) CountAllEnabledNodeClusters(ctx context.Context, req *pb.CountAllEnabledNodeClustersRequest) (*pb.CountAllEnabledNodeClustersResponse, error) {
	_, _, err := rpcutils.ValidateRequest(ctx, rpcutils.UserTypeAdmin)
	if err != nil {
		return nil, err
	}

	count, err := models.SharedNodeClusterDAO.CountAllEnabledClusters()
	if err != nil {
		return nil, err
	}

	return &pb.CountAllEnabledNodeClustersResponse{Count: count}, nil
}

// 列出单页集群
func (this *NodeClusterService) ListEnabledNodeClusters(ctx context.Context, req *pb.ListEnabledNodeClustersRequest) (*pb.ListEnabledNodeClustersResponse, error) {
	_, _, err := rpcutils.ValidateRequest(ctx, rpcutils.UserTypeAdmin)
	if err != nil {
		return nil, err
	}

	clusters, err := models.SharedNodeClusterDAO.ListEnabledClusters(req.Offset, req.Size)
	if err != nil {
		return nil, err
	}

	result := []*pb.NodeCluster{}
	for _, cluster := range clusters {
		result = append(result, &pb.NodeCluster{
			Id:         int64(cluster.Id),
			Name:       cluster.Name,
			CreatedAt:  int64(cluster.CreatedAt),
			GrantId:    int64(cluster.GrantId),
			InstallDir: cluster.InstallDir,
			UniqueId:   cluster.UniqueId,
			Secret:     cluster.Secret,
		})
	}

	return &pb.ListEnabledNodeClustersResponse{Clusters: result}, nil
}

// 查找集群的健康检查配置
func (this *NodeClusterService) FindNodeClusterHealthCheckConfig(ctx context.Context, req *pb.FindNodeClusterHealthCheckConfigRequest) (*pb.FindNodeClusterHealthCheckConfigResponse, error) {
	// 校验请求
	_, _, err := rpcutils.ValidateRequest(ctx, rpcutils.UserTypeAdmin)
	if err != nil {
		return nil, err
	}

	config, err := models.SharedNodeClusterDAO.FindClusterHealthCheckConfig(req.ClusterId)
	if err != nil {
		return nil, err
	}
	configJSON, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}
	return &pb.FindNodeClusterHealthCheckConfigResponse{HealthCheckConfig: configJSON}, nil
}

// 修改集群健康检查设置
func (this *NodeClusterService) UpdateNodeClusterHealthCheck(ctx context.Context, req *pb.UpdateNodeClusterHealthCheckRequest) (*pb.RPCUpdateSuccess, error) {
	// 校验请求
	_, _, err := rpcutils.ValidateRequest(ctx, rpcutils.UserTypeAdmin)
	if err != nil {
		return nil, err
	}

	err = models.SharedNodeClusterDAO.UpdateClusterHealthCheck(req.ClusterId, req.HealthCheckJSON)
	if err != nil {
		return nil, err
	}
	return rpcutils.RPCUpdateSuccess()
}

// 执行健康检查
func (this *NodeClusterService) ExecuteNodeClusterHealthCheck(ctx context.Context, req *pb.ExecuteNodeClusterHealthCheckRequest) (*pb.ExecuteNodeClusterHealthCheckResponse, error) {
	// 校验请求
	_, _, err := rpcutils.ValidateRequest(ctx, rpcutils.UserTypeAdmin)
	if err != nil {
		return nil, err
	}

	executor := tasks.NewHealthCheckExecutor(req.ClusterId)
	results, err := executor.Run()
	if err != nil {
		return nil, err
	}
	pbResults := []*pb.ExecuteNodeClusterHealthCheckResponse_Result{}
	for _, result := range results {
		pbResults = append(pbResults, &pb.ExecuteNodeClusterHealthCheckResponse_Result{
			Node: &pb.Node{
				Id:   int64(result.Node.Id),
				Name: result.Node.Name,
			},
			NodeAddr: result.NodeAddr,
			IsOk:     result.IsOk,
			Error:    result.Error,
			CostMs:   types.Float32(result.CostMs),
		})
	}
	return &pb.ExecuteNodeClusterHealthCheckResponse{Results: pbResults}, nil
}

// 计算使用某个认证的集群数量
func (this *NodeClusterService) CountAllEnabledNodeClustersWithGrantId(ctx context.Context, req *pb.CountAllEnabledNodeClustersWithGrantIdRequest) (*pb.CountAllEnabledNodeClustersWithGrantIdResponse, error) {
	// 校验请求
	_, _, err := rpcutils.ValidateRequest(ctx, rpcutils.UserTypeAdmin)
	if err != nil {
		return nil, err
	}

	count, err := models.SharedNodeClusterDAO.CountAllEnabledClustersWithGrantId(req.GrantId)
	if err != nil {
		return nil, err
	}
	return &pb.CountAllEnabledNodeClustersWithGrantIdResponse{Count: count}, nil
}

// 查找使用某个认证的所有集群
func (this *NodeClusterService) FindAllEnabledNodeClustersWithGrantId(ctx context.Context, req *pb.FindAllEnabledNodeClustersWithGrantIdRequest) (*pb.FindAllEnabledNodeClustersWithGrantIdResponse, error) {
	// 校验请求
	_, _, err := rpcutils.ValidateRequest(ctx, rpcutils.UserTypeAdmin)
	if err != nil {
		return nil, err
	}

	clusters, err := models.SharedNodeClusterDAO.FindAllEnabledClustersWithGrantId(req.GrantId)
	if err != nil {
		return nil, err
	}

	result := []*pb.NodeCluster{}
	for _, cluster := range clusters {
		result = append(result, &pb.NodeCluster{
			Id:        int64(cluster.Id),
			Name:      cluster.Name,
			CreatedAt: int64(cluster.CreatedAt),
			UniqueId:  cluster.UniqueId,
			Secret:    cluster.Secret,
		})
	}
	return &pb.FindAllEnabledNodeClustersWithGrantIdResponse{Clusters: result}, nil
}

// 查找集群的DNS配置
func (this *NodeClusterService) FindEnabledNodeClusterDNS(ctx context.Context, req *pb.FindEnabledNodeClusterDNSRequest) (*pb.FindEnabledNodeClusterDNSResponse, error) {
	// 校验请求
	_, _, err := rpcutils.ValidateRequest(ctx, rpcutils.UserTypeAdmin)
	if err != nil {
		return nil, err
	}

	dnsInfo, err := models.SharedNodeClusterDAO.FindClusterDNSInfo(req.NodeClusterId)
	if err != nil {
		return nil, err
	}
	if dnsInfo == nil {
		return &pb.FindEnabledNodeClusterDNSResponse{
			Name:     "",
			Domain:   "",
			Provider: nil,
		}, nil
	}

	provider, err := models.SharedDNSProviderDAO.FindEnabledDNSProvider(int64(dnsInfo.DnsProviderId))
	if err != nil {
		return nil, err
	}

	var pbProvider *pb.DNSProvider = nil
	if provider != nil {
		pbProvider = &pb.DNSProvider{
			Id:       int64(provider.Id),
			Type:     provider.Type,
			TypeName: dnsproviders.FindProviderTypeName(provider.Type),
		}
	}

	return &pb.FindEnabledNodeClusterDNSResponse{
		Name:     dnsInfo.DnsName,
		Domain:   dnsInfo.DnsDomain,
		Provider: pbProvider,
	}, nil
}

// 计算使用某个DNS服务商的集群数量
func (this *NodeClusterService) CountAllEnabledNodeClustersWithDNSProviderId(ctx context.Context, req *pb.CountAllEnabledNodeClustersWithDNSProviderIdRequest) (*pb.CountAllEnabledNodeClustersWithDNSProviderIdResponse, error) {
	// 校验请求
	_, _, err := rpcutils.ValidateRequest(ctx, rpcutils.UserTypeAdmin)
	if err != nil {
		return nil, err
	}

	count, err := models.SharedNodeClusterDAO.CountAllEnabledClustersWithDNSProviderId(req.DnsProviderId)
	if err != nil {
		return nil, err
	}
	return &pb.CountAllEnabledNodeClustersWithDNSProviderIdResponse{Count: count}, nil
}
