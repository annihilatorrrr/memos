package v1

import (
	"context"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	v1pb "github.com/usememos/memos/proto/gen/api/v1"
	storepb "github.com/usememos/memos/proto/gen/store"
	"github.com/usememos/memos/store"
)

func (s *APIV1Service) ListWorkspaceSettings(ctx context.Context, _ *v1pb.ListWorkspaceSettingsRequest) (*v1pb.ListWorkspaceSettingsResponse, error) {
	workspaceSettings, err := s.Store.ListWorkspaceSettings(ctx, &store.FindWorkspaceSetting{})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get workspace setting: %v", err)
	}

	response := &v1pb.ListWorkspaceSettingsResponse{
		Settings: []*v1pb.WorkspaceSetting{},
	}
	for _, workspaceSetting := range workspaceSettings {
		if workspaceSetting.Key == storepb.WorkspaceSettingKey_WORKSPACE_SETTING_BASIC {
			continue
		}
		response.Settings = append(response.Settings, convertWorkspaceSettingFromStore(workspaceSetting))
	}
	return response, nil
}

func (s *APIV1Service) GetWorkspaceSetting(ctx context.Context, request *v1pb.GetWorkspaceSettingRequest) (*v1pb.WorkspaceSetting, error) {
	settingKeyString, err := ExtractWorkspaceSettingKeyFromName(request.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid workspace setting name: %v", err)
	}
	settingKey := storepb.WorkspaceSettingKey(storepb.WorkspaceSettingKey_value[settingKeyString])
	workspaceSetting, err := s.Store.GetWorkspaceSetting(ctx, &store.FindWorkspaceSetting{
		Name: settingKey.String(),
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get workspace setting: %v", err)
	}
	if workspaceSetting == nil {
		return nil, status.Errorf(codes.NotFound, "workspace setting not found")
	}

	return convertWorkspaceSettingFromStore(workspaceSetting), nil
}

func (s *APIV1Service) SetWorkspaceSetting(ctx context.Context, request *v1pb.SetWorkspaceSettingRequest) (*v1pb.WorkspaceSetting, error) {
	if s.Profile.Mode == "demo" {
		return nil, status.Errorf(codes.InvalidArgument, "setting workspace setting is not allowed in demo mode")
	}

	user, err := getCurrentUser(ctx, s.Store)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get current user: %v", err)
	}
	if user.Role != store.RoleHost {
		return nil, status.Errorf(codes.PermissionDenied, "permission denied")
	}

	workspaceSetting, err := s.Store.UpsertWorkspaceSetting(ctx, convertWorkspaceSettingToStore(request.Setting))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to upsert workspace setting: %v", err)
	}

	return convertWorkspaceSettingFromStore(workspaceSetting), nil
}

func convertWorkspaceSettingFromStore(setting *storepb.WorkspaceSetting) *v1pb.WorkspaceSetting {
	workspaceSetting := &v1pb.WorkspaceSetting{
		Name: fmt.Sprintf("%s%s", WorkspaceSettingNamePrefix, setting.Key.String()),
	}
	switch setting.Value.(type) {
	case *storepb.WorkspaceSetting_GeneralSetting:
		workspaceSetting.Value = &v1pb.WorkspaceSetting_GeneralSetting{
			GeneralSetting: convertWorkspaceGeneralSettingFromStore(setting.GetGeneralSetting()),
		}
	case *storepb.WorkspaceSetting_StorageSetting:
		workspaceSetting.Value = &v1pb.WorkspaceSetting_StorageSetting{
			StorageSetting: convertWorkspaceStorageSettingFromStore(setting.GetStorageSetting()),
		}
	case *storepb.WorkspaceSetting_MemoRelatedSetting:
		workspaceSetting.Value = &v1pb.WorkspaceSetting_MemoRelatedSetting{
			MemoRelatedSetting: convertWorkspaceMemoRelatedSettingFromStore(setting.GetMemoRelatedSetting()),
		}
	}
	return workspaceSetting
}

func convertWorkspaceSettingToStore(setting *v1pb.WorkspaceSetting) *storepb.WorkspaceSetting {
	settingKeyString, _ := ExtractWorkspaceSettingKeyFromName(setting.Name)
	workspaceSetting := &storepb.WorkspaceSetting{
		Key: storepb.WorkspaceSettingKey(storepb.WorkspaceSettingKey_value[settingKeyString]),
		Value: &storepb.WorkspaceSetting_GeneralSetting{
			GeneralSetting: convertWorkspaceGeneralSettingToStore(setting.GetGeneralSetting()),
		},
	}
	switch workspaceSetting.Key {
	case storepb.WorkspaceSettingKey_WORKSPACE_SETTING_GENERAL:
		workspaceSetting.Value = &storepb.WorkspaceSetting_GeneralSetting{
			GeneralSetting: convertWorkspaceGeneralSettingToStore(setting.GetGeneralSetting()),
		}
	case storepb.WorkspaceSettingKey_WORKSPACE_SETTING_STORAGE:
		workspaceSetting.Value = &storepb.WorkspaceSetting_StorageSetting{
			StorageSetting: convertWorkspaceStorageSettingToStore(setting.GetStorageSetting()),
		}
	case storepb.WorkspaceSettingKey_WORKSPACE_SETTING_MEMO_RELATED:
		workspaceSetting.Value = &storepb.WorkspaceSetting_MemoRelatedSetting{
			MemoRelatedSetting: convertWorkspaceMemoRelatedSettingToStore(setting.GetMemoRelatedSetting()),
		}
	}
	return workspaceSetting
}

func convertWorkspaceGeneralSettingFromStore(setting *storepb.WorkspaceGeneralSetting) *v1pb.WorkspaceGeneralSetting {
	if setting == nil {
		return nil
	}
	generalSetting := &v1pb.WorkspaceGeneralSetting{
		InstanceUrl:           setting.InstanceUrl,
		DisallowSignup:        setting.DisallowSignup,
		DisallowPasswordLogin: setting.DisallowPasswordLogin,
		AdditionalScript:      setting.AdditionalScript,
		AdditionalStyle:       setting.AdditionalStyle,
	}
	if setting.CustomProfile != nil {
		generalSetting.CustomProfile = &v1pb.WorkspaceCustomProfile{
			Title:       setting.CustomProfile.Title,
			Description: setting.CustomProfile.Description,
			LogoUrl:     setting.CustomProfile.LogoUrl,
			Locale:      setting.CustomProfile.Locale,
			Appearance:  setting.CustomProfile.Appearance,
		}
	}
	return generalSetting
}

func convertWorkspaceGeneralSettingToStore(setting *v1pb.WorkspaceGeneralSetting) *storepb.WorkspaceGeneralSetting {
	if setting == nil {
		return nil
	}
	generalSetting := &storepb.WorkspaceGeneralSetting{
		InstanceUrl:           setting.InstanceUrl,
		DisallowSignup:        setting.DisallowSignup,
		DisallowPasswordLogin: setting.DisallowPasswordLogin,
		AdditionalScript:      setting.AdditionalScript,
		AdditionalStyle:       setting.AdditionalStyle,
	}
	if setting.CustomProfile != nil {
		generalSetting.CustomProfile = &storepb.WorkspaceCustomProfile{
			Title:       setting.CustomProfile.Title,
			Description: setting.CustomProfile.Description,
			LogoUrl:     setting.CustomProfile.LogoUrl,
			Locale:      setting.CustomProfile.Locale,
			Appearance:  setting.CustomProfile.Appearance,
		}
	}
	return generalSetting
}

func convertWorkspaceStorageSettingFromStore(setting *storepb.WorkspaceStorageSetting) *v1pb.WorkspaceStorageSetting {
	if setting == nil {
		return nil
	}
	return &v1pb.WorkspaceStorageSetting{
		StorageType:              v1pb.WorkspaceStorageSetting_StorageType(setting.StorageType),
		LocalStoragePathTemplate: setting.LocalStoragePathTemplate,
		UploadSizeLimitMb:        setting.UploadSizeLimitMb,
		ActivedExternalStorageId: setting.ActivedExternalStorageId,
	}
}

func convertWorkspaceStorageSettingToStore(setting *v1pb.WorkspaceStorageSetting) *storepb.WorkspaceStorageSetting {
	if setting == nil {
		return nil
	}
	return &storepb.WorkspaceStorageSetting{
		StorageType:              storepb.WorkspaceStorageSetting_StorageType(setting.StorageType),
		LocalStoragePathTemplate: setting.LocalStoragePathTemplate,
		UploadSizeLimitMb:        setting.UploadSizeLimitMb,
		ActivedExternalStorageId: setting.ActivedExternalStorageId,
	}
}

func convertWorkspaceMemoRelatedSettingFromStore(setting *storepb.WorkspaceMemoRelatedSetting) *v1pb.WorkspaceMemoRelatedSetting {
	if setting == nil {
		return nil
	}
	return &v1pb.WorkspaceMemoRelatedSetting{
		DisallowPublicVisible: setting.DisallowPublicVisible,
		DisplayWithUpdateTime: setting.DisplayWithUpdateTime,
	}
}

func convertWorkspaceMemoRelatedSettingToStore(setting *v1pb.WorkspaceMemoRelatedSetting) *storepb.WorkspaceMemoRelatedSetting {
	if setting == nil {
		return nil
	}
	return &storepb.WorkspaceMemoRelatedSetting{
		DisallowPublicVisible: setting.DisallowPublicVisible,
		DisplayWithUpdateTime: setting.DisplayWithUpdateTime,
	}
}
