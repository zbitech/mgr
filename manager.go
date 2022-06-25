package mgr

import (
	"context"
	"errors"
	"github.com/zbitech/common/pkg/logger"
	"github.com/zbitech/common/pkg/vars"

	"github.com/zbitech/common/interfaces"
	"github.com/zbitech/common/pkg/errs"
	"github.com/zbitech/common/pkg/model/ztypes"
	"github.com/zbitech/mgr/rsc"
)

type ResourceManagerFactory struct {
	ingressManager     interfaces.AppResourceManagerIF
	projectManager     interfaces.ProjectResourceManagerIF
	zcashManager       interfaces.InstanceResourceManagerIF
	lightWalletManager interfaces.InstanceResourceManagerIF
}

func NewResourceManagerFactory() interfaces.ResourceManagerFactoryIF {
	return &ResourceManagerFactory{}
}

func (m *ResourceManagerFactory) Init(ctx context.Context) error {

	var err error

	zcashCfg, ok := vars.ResourceConfig.GetInstanceResourceConfig(ztypes.InstanceTypeZCASH)
	if !ok {
		return errors.New("unable to retrieve zcash configuration")
	}

	m.zcashManager, err = rsc.NewZcashInstanceResourceManager(zcashCfg)
	if err != nil {
		logger.Errorf(ctx, "Failed to create zcash resource manager - %s", err)
		return errs.ErrInstanceResourceFailed
	}

	lwdCfg, ok := vars.ResourceConfig.GetInstanceResourceConfig(ztypes.InstanceTypeLWD)
	if !ok {
		return errors.New("unable to retrieve lightwalletd configuration")
	}

	m.lightWalletManager, err = rsc.NewLWDInstanceResourceManager(lwdCfg)
	if err != nil {
		logger.Errorf(ctx, "Failed to create lightwalletd resource manager - %s", err)
		return errs.ErrInstanceResourceFailed
	}

	m.projectManager, err = rsc.NewProjectResourceManager(vars.ResourceConfig.Project,
		map[ztypes.InstanceType]interfaces.InstanceResourceManagerIF{
			ztypes.InstanceTypeZCASH: m.zcashManager,
			ztypes.InstanceTypeLWD:   m.lightWalletManager,
		})
	if err != nil {
		logger.Errorf(ctx, "Failed to create project resource manager - %s", err)
		return errs.ErrProjectResourceFailed
	}

	logger.Infof(ctx, "Initializing App Manager with %v and %v", vars.ResourceConfig.App, m.projectManager)
	m.ingressManager, err = rsc.NewAppResourceManager(vars.ResourceConfig.App, m.projectManager)
	if err != nil {
		logger.Errorf(ctx, "Failed to create ingress resource manager - %s", err)
		return errs.ErrIngressResourceFailed
	}

	return nil
}

func (m *ResourceManagerFactory) GetAppResourceManager(ctx context.Context) interfaces.AppResourceManagerIF {
	return m.ingressManager
}

func (m *ResourceManagerFactory) GetProjectDataManager(ctx context.Context) interfaces.ProjectResourceManagerIF {
	return m.projectManager
}
