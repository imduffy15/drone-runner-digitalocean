// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Polyform License
// that can be found in the LICENSE file.

// Package platform contains code to provision and destroy server
// instances on the Digital Ocean cloud platform.
package platform

import (
	"context"
	"fmt"
	"time"

	"github.com/drone/runner-go/logger"
	compute "google.golang.org/api/compute/v1"
)

type (
	// DestroyArgs provides arguments to destroy the server
	// instance.
	DestroyArgs struct {
		ID        string
		Region    string
		ProjectID string
	}

	// ProvisionArgs provides arguments to provision instances.
	ProvisionArgs struct {
		Key       string
		Image     string
		Name      string
		Region    string
		Size      string
		ProjectID string
		User      string
	}

	// Instance represents a provisioned server instance.
	Instance struct {
		ID string
		IP string
	}
)

// Provision provisions the server instance.
func Provision(ctx context.Context, args ProvisionArgs) (Instance, error) {
	res := Instance{}

	prefix := "https://www.googleapis.com/compute/v1/projects/" + args.ProjectID

	sshKey := fmt.Sprintf("%s:%s", args.User, args.Key)

	bool := "FALSE"

	req := &compute.Instance{
		Name:        args.Name,
		Description: "Drone builder instance",
		MachineType: prefix + "/zones/" + args.Region + "/machineTypes/" + args.Size,
		Disks: []*compute.AttachedDisk{
			{
				AutoDelete: true,
				Boot:       true,
				Type:       "PERSISTENT",
				InitializeParams: &compute.AttachedDiskInitializeParams{
					SourceImage: args.Image,
				},
			},
		},
		NetworkInterfaces: []*compute.NetworkInterface{
			{
				AccessConfigs: []*compute.AccessConfig{
					{
						Type: "ONE_TO_ONE_NAT",
						Name: "External NAT",
					},
				},
				Network: prefix + "/global/networks/default",
			},
		},
		ServiceAccounts: []*compute.ServiceAccount{
			{
				Email: "default",
				Scopes: []string{
					compute.DevstorageFullControlScope,
					compute.ComputeScope,
				},
			},
		},
		Metadata: &compute.Metadata{
			Items: []*compute.MetadataItems{
				&compute.MetadataItems{
					Key:   "ssh-keys",
					Value: &sshKey,
				},
				&compute.MetadataItems{
					Key:   "enable-oslogin",
					Value: &bool,
				},
			},
		},
	}

	logger := logger.FromContext(ctx).
		WithField("region", args.Region).
		WithField("image", args.Image).
		WithField("size", args.Size).
		WithField("name", args.Name).
		WithField("project_id", args.ProjectID)

	logger.Debug("instance create")

	client, err := compute.NewService(ctx)
	if err != nil {
		logger.WithError(err).Error("unable to create compute service")
		return res, err
	}
	_, err = client.Instances.Insert(args.ProjectID, args.Region, req).Do()
	if err != nil {
		logger.WithError(err).Error("cannot create instance")
		return res, err
	}

	// record the droplet ID
	res.ID = args.Name

	logger.WithField("name", args.Name).
		Info("instance created")

	// poll the gcp endpoint for server updates
	// and exit when a network address is allocated and ssh is available.
	interval := time.Duration(0)
poller:
	for {
		select {
		case <-ctx.Done():
			logger.WithField("name", args.Name).
				Debug("cannot ascertain network")

			return res, ctx.Err()
		case <-time.After(interval):
			interval = time.Second * 30

			logger.WithField("name", args.Name).
				Debug("find instance network")

			inst, err := client.Instances.Get(args.ProjectID, args.Region, args.Name).Do()
			if err != nil {
				logger.WithError(err).
					Error("cannot find instance")
				return res, err
			}

			for _, iface := range inst.NetworkInterfaces {
				for _, accessConfig := range iface.AccessConfigs {
					if accessConfig.Type == "ONE_TO_ONE_NAT" {
						res.IP = accessConfig.NatIP
					}
				}
			}

			if res.IP != "" {
				break poller
			}
		}
	}

	logger.WithField("name", args.Name).
		WithField("ip", res.IP).
		WithField("id", res.ID).
		Debug("instance network ready")

	return res, nil
}

// Destroy destroys the server instance.
func Destroy(ctx context.Context, args DestroyArgs) error {
	logger := logger.FromContext(ctx).
		WithField("project_id", args.ProjectID).
		WithField("region", args.Region).WithField("id", args.ID)

	client, err := compute.NewService(ctx)
	if err != nil {
		logger.WithError(err).Error("unable to create compute service")
		return err
	}
	_, err = client.Instances.Delete(args.ProjectID, args.Region, args.ID).Do()
	if err != nil {
		logger.WithError(err).Error("Unable to delete instance")
		return err
	}
	return nil
}
