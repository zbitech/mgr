package mgr

import (
	"context"

	"github.com/zbitech/common/interfaces"
	"github.com/zbitech/common/pkg/errs"
	"github.com/zbitech/common/pkg/model/ztypes"
	"github.com/zbitech/mgr/rsc"
)

type ResourceManagerFactory struct {
	projectManager     interfaces.ProjectResourceManagerIF
	zcashManager       interfaces.InstanceResourceManagerIF
	lightWalletManager interfaces.InstanceResourceManagerIF
}

func NewResourceManagerFactory() interfaces.ResourceManagerFactoryIF {
	return &ResourceManagerFactory{}
}

func (m *ResourceManagerFactory) Init(ctx context.Context) error {

	var err error

	m.zcashManager, err = rsc.NewZcashInstanceResourceManager()
	if err != nil {
		return errs.ErrInstanceResourceFailed
	}

	m.lightWalletManager, err = rsc.NewLWDInstanceResourceManager()
	if err != nil {
		return errs.ErrInstanceResourceFailed
	}

	m.projectManager, err = rsc.NewProjectResourceManager()
	if err != nil {
		return errs.ErrProjectResourceFailed
	}

	m.projectManager.AddInstanceManager(ctx, ztypes.ZCASH_INSTANCE, m.zcashManager)
	m.projectManager.AddInstanceManager(ctx, ztypes.LWD_INSTANCE, m.lightWalletManager)

	return nil
}

func (m *ResourceManagerFactory) GetProjectDataManager(ctx context.Context) interfaces.ProjectResourceManagerIF {
	return m.projectManager
}
