package server

import (
	"context"
	"github.com/bootapp/rest-grpc-oauth2/auth"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"log"
	core "srv-core/proto"
	"srv-core/settings"
)

type SrvCoreAuthServiceServer struct {
	dalCoreUserClient core.DalUserServiceClient
	dalCoreUserConn *grpc.ClientConn
	dalCoreAuthClient core.DalAuthServiceClient
	dalCoreAuthConn *grpc.ClientConn
	auth *auth.StatelessAuthenticator
}

func NewSysServer() *SrvCoreAuthServiceServer {
	s := &SrvCoreAuthServiceServer{}
	var err error
	s.dalCoreUserConn, err = grpc.Dial(settings.DalCoreUserAddr, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	s.dalCoreUserClient = core.NewDalUserServiceClient(s.dalCoreUserConn)

	s.dalCoreAuthConn, err = grpc.Dial(settings.DalCoreSysAddr, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	s.dalCoreAuthClient = core.NewDalAuthServiceClient(s.dalCoreUserConn)

	s.auth = auth.GetInstance()
	return s
}

func (s *SrvCoreAuthServiceServer) InvokeUpdateAuthGroups(ctx context.Context, req *core.InvokeUpdateAuthGroupReq) (*core.Empty, error) {
	userId, orgId := s.auth.GetAuthInfo(ctx)
	if userId == 0 || orgId == 0 {
		return nil, status.Error(codes.Unauthenticated, "UNAUTHENTICATED")
	}
	return s.dalCoreAuthClient.UpdateAuthGroups(ctx, &core.AuthGroupsReq{UserId:userId, OrgId:orgId, Data:req.Data})
}

func (s *SrvCoreAuthServiceServer) InvokeDeleteAuthGroup(ctx context.Context, req *core.IdReq) (*core.Empty, error) {
	userId, orgId := s.auth.GetAuthInfo(ctx)
	if userId == 0 || orgId == 0 {
		return nil, status.Error(codes.Unauthenticated, "UNAUTHENTICATED")
	}
	return s.dalCoreAuthClient.DeleteAuthGroups(ctx, &core.AuthorizedIdsReq{UserId:userId, OrgId:orgId, Ids:[]int64{req.Id}})
}

func (s *SrvCoreAuthServiceServer) QueryAuthGroups(ctx context.Context, req *core.QueryAuthGroupReq) (*core.AuthGroupsResp, error) {
	userId, orgId := s.auth.GetAuthInfo(ctx)
	if userId == 0 || orgId == 0 {
		return nil, status.Error(codes.Unauthenticated, "UNAUTHENTICATED")
	}
	return s.dalCoreAuthClient.ReadAuthGroups(ctx, &core.ReadAuthGroupsReq{UserId:userId, OrgId:orgId, Pid:req.Pid})
}

func (s *SrvCoreAuthServiceServer) InvokeNewAuthority(ctx context.Context, req *core.AuthorityEdit) (*core.Empty, error) {
	userId, orgId := s.auth.GetAuthInfo(ctx)
	if userId == 0 || orgId == 0 {
		return nil, status.Error(codes.Unauthenticated, "UNAUTHENTICATED")
	}
	return s.dalCoreAuthClient.CreateAuthorities(ctx, &core.AuthoritiesReq{UserId:userId, OrgId:orgId, Data:[]*core.AuthorityEdit{req}})

}

func (s *SrvCoreAuthServiceServer) InvokeUpdateAuthority(ctx context.Context, req *core.AuthorityEdit) (*core.Empty, error) {
	userId, orgId := s.auth.GetAuthInfo(ctx)
	if userId == 0 || orgId == 0 {
		return nil, status.Error(codes.Unauthenticated, "UNAUTHENTICATED")
	}
	return s.dalCoreAuthClient.UpdateAuthorities(ctx, &core.AuthoritiesReq{UserId:userId, OrgId:orgId, Data:[]*core.AuthorityEdit{req}})

}

func (s *SrvCoreAuthServiceServer) InvokeDeleteAuthority(ctx context.Context, req *core.InvokeDeleteAuthReq) (*core.Empty, error) {
	userId, orgId := s.auth.GetAuthInfo(ctx)
	if userId == 0 || orgId == 0 {
		return nil, status.Error(codes.Unauthenticated, "UNAUTHENTICATED")
	}
	return s.dalCoreAuthClient.DeleteAuthorities(ctx, &core.DeleteAuthoritiesReq{UserId:userId, OrgId:orgId, Data:[]string{req.Key}})
}

func (s *SrvCoreAuthServiceServer) QueryAuthority(ctx context.Context, req *core.QueryAuthReq) (*core.AuthoritiesResp, error) {
	userId, orgId := s.auth.GetAuthInfo(ctx)
	if userId == 0 || orgId == 0 {
		return nil, status.Error(codes.Unauthenticated, "UNAUTHENTICATED")
	}
	return s.dalCoreAuthClient.ReadAuthorities(ctx, &core.ReadAuthoritiesReq{UserId:userId, OrgId:orgId, GroupId:req.GroupId})
}

func (s *SrvCoreAuthServiceServer) InvokeUpdateRoleOrgs(ctx context.Context, req *core.InvokeUpdateRoleOrgsReq) (*core.Empty, error) {
	userId, orgId := s.auth.GetAuthInfo(ctx)
	if userId == 0 || orgId == 0 {
		return nil, status.Error(codes.Unauthenticated, "UNAUTHENTICATED")
	}
	return s.dalCoreAuthClient.UpdateRoleOrgs(ctx, &core.RoleOrgsReq{UserId:userId, OrgId:orgId, Data:req.Data})
}

func (s *SrvCoreAuthServiceServer) InvokeDeleteRoleOrg(ctx context.Context, req *core.IdReq) (*core.Empty, error) {
	userId, orgId := s.auth.GetAuthInfo(ctx)
	if userId == 0 || orgId == 0 {
		return nil, status.Error(codes.Unauthenticated, "UNAUTHENTICATED")
	}
	return s.dalCoreAuthClient.DeleteRoleOrgs(ctx, &core.AuthorizedIdsReq{UserId:userId, OrgId:orgId, Ids:[]int64{req.Id}})
}

func (s *SrvCoreAuthServiceServer) QueryRoleOrgs(ctx context.Context, req *core.PaginationReq) (*core.RoleOrgsResp, error) {
	userId, orgId := s.auth.GetAuthInfo(ctx)
	if userId == 0 || orgId == 0 {
		return nil, status.Error(codes.Unauthenticated, "UNAUTHENTICATED")
	}
	return s.dalCoreAuthClient.ReadRoleOrgs(ctx, &core.AuthorizedPaginationReq{UserId:userId, OrgId:orgId, Pagination:req.Pagination})
}

func (s *SrvCoreAuthServiceServer) InvokeUpdateRoleUsers(ctx context.Context, req *core.InvokeUpdateRoleUsersReq) (*core.Empty, error) {
	userId, orgId := s.auth.GetAuthInfo(ctx)
	if userId == 0 || orgId == 0 {
		return nil, status.Error(codes.Unauthenticated, "UNAUTHENTICATED")
	}
	return s.dalCoreAuthClient.UpdateRoleUsers(ctx, &core.RoleUsersReq{UserId:userId, OrgId:orgId, Data:req.Data})
}

func (s *SrvCoreAuthServiceServer) InvokeDeleteRoleUser(ctx context.Context, req *core.IdReq) (*core.Empty, error) {
	userId, orgId := s.auth.GetAuthInfo(ctx)
	if userId == 0 || orgId == 0 {
		return nil, status.Error(codes.Unauthenticated, "UNAUTHENTICATED")
	}
	return s.dalCoreAuthClient.DeleteRoleUsers(ctx, &core.AuthorizedIdsReq{UserId:userId, OrgId:orgId, Ids:[]int64{req.Id}})
}

func (s *SrvCoreAuthServiceServer) QueryRoleUsers(ctx context.Context, req *core.PaginationReq) (*core.RoleUsersResp, error) {
	userId, orgId := s.auth.GetAuthInfo(ctx)
	if userId == 0 || orgId == 0 {
		return nil, status.Error(codes.Unauthenticated, "UNAUTHENTICATED")
	}
	return s.dalCoreAuthClient.ReadRoleUsers(ctx, &core.AuthorizedPaginationReq{UserId:userId, OrgId:orgId, Pagination:req.Pagination})
}