package controller

import (
	"fmt"
	"github.com/lf-edge/eden/pkg/utils"
	"github.com/lf-edge/eve/api/go/config"
)

//GetContentTree return ContentTree config from cloud by ID
func (cloud *CloudCtx) GetContentTree(id string) (contentTree *config.ContentTree, err error) {
	for _, contentTree := range cloud.contentTrees {
		if contentTree.Uuid == id {
			return contentTree, nil
		}
	}
	return nil, fmt.Errorf("not found ContentTree with ID: %s", id)
}

//AddContentTree add ContentTree config to cloud
func (cloud *CloudCtx) AddContentTree(contentTree *config.ContentTree) error {
	for _, tree := range cloud.contentTrees {
		if tree.Uuid == contentTree.GetUuid() {
			return fmt.Errorf("ContentTree already exists with ID: %s", contentTree.GetUuid())
		}
	}
	cloud.contentTrees = append(cloud.contentTrees, contentTree)
	return nil
}

//RemoveContentTree remove ContentTree config to cloud
func (cloud *CloudCtx) RemoveContentTree(id string) error {
	for ind, contentTree := range cloud.contentTrees {
		if contentTree.Uuid == id {
			utils.DelEleInSlice(&cloud.contentTrees, ind)
			return nil
		}
	}
	return fmt.Errorf("not found ContentTree with ID: %s", id)
}

//ListContentTree return ContentTree configs from cloud
func (cloud *CloudCtx) ListContentTree() []*config.ContentTree {
	return cloud.contentTrees
}
