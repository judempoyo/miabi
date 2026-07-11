// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package repositories

import (
	"github.com/miabi-io/miabi/internal/models"
	"gorm.io/gorm"
)

type NetworkAllocationRepository struct {
	db *gorm.DB
}

func NewNetworkAllocationRepository(db *gorm.DB) *NetworkAllocationRepository {
	return &NetworkAllocationRepository{db: db}
}

// Create reserves an allocation. The unique indexes on docker_name and subnet
// make concurrent allocation of the same subnet fail for all but one caller.
func (r *NetworkAllocationRepository) Create(a *models.NetworkAllocation) error {
	return r.db.Create(a).Error
}

// FindByDockerName returns the allocation pinned to a Docker network name, or
// gorm.ErrRecordNotFound.
func (r *NetworkAllocationRepository) FindByDockerName(dockerName string) (*models.NetworkAllocation, error) {
	var a models.NetworkAllocation
	if err := r.db.Where("docker_name = ?", dockerName).First(&a).Error; err != nil {
		return nil, err
	}
	return &a, nil
}

// DeleteByDockerName frees an allocation when its network is removed.
func (r *NetworkAllocationRepository) DeleteByDockerName(dockerName string) error {
	return r.db.Where("docker_name = ?", dockerName).Delete(&models.NetworkAllocation{}).Error
}

// ExistsBySubnet reports whether a subnet is already allocated or reserved.
func (r *NetworkAllocationRepository) ExistsBySubnet(subnet string) (bool, error) {
	var n int64
	err := r.db.Model(&models.NetworkAllocation{}).Where("subnet = ?", subnet).Count(&n).Error
	return n > 0, err
}

// AllSubnets returns every allocated/reserved subnet — the used-set the allocator
// scans against to find the next free one.
func (r *NetworkAllocationRepository) AllSubnets() ([]string, error) {
	var subnets []string
	err := r.db.Model(&models.NetworkAllocation{}).Pluck("subnet", &subnets).Error
	return subnets, err
}

// Count returns how many subnets are allocated or reserved (for pool-utilization
// metrics). Reserved rows are included since they consume pool capacity.
func (r *NetworkAllocationRepository) Count() (int64, error) {
	var n int64
	err := r.db.Model(&models.NetworkAllocation{}).Count(&n).Error
	return n, err
}
