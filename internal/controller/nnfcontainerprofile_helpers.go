/*
 * Copyright 2023 Hewlett Packard Enterprise Development LP
 * Other additional copyright holders may be indicated within.
 *
 * The entirety of this work is licensed under the Apache License,
 * Version 2.0 (the "License"); you may not use this file except
 * in compliance with the License.
 *
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package controller

import (
	"context"
	"fmt"
	"os"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	dwsv1alpha2 "github.com/DataWorkflowServices/dws/api/v1alpha2"
	"github.com/DataWorkflowServices/dws/utils/dwdparse"
	nnfv1alpha1 "github.com/NearNodeFlash/nnf-sos/api/v1alpha1"
	"github.com/go-logr/logr"
)

func getContainerProfile(ctx context.Context, clnt client.Client, workflow *dwsv1alpha2.Workflow, index int) (*nnfv1alpha1.NnfContainerProfile, error) {
	profile, err := findPinnedContainerProfile(ctx, clnt, workflow, index)
	if err != nil {
		return nil, err
	}

	if profile == nil {
		return nil, dwsv1alpha2.NewResourceError("container profile '%s' not found", indexedResourceName(workflow, index)).WithFatal()
	}

	return profile, nil
}

func findPinnedContainerProfile(ctx context.Context, clnt client.Client, workflow *dwsv1alpha2.Workflow, index int) (*nnfv1alpha1.NnfContainerProfile, error) {
	profile := &nnfv1alpha1.NnfContainerProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      indexedResourceName(workflow, index),
			Namespace: workflow.Namespace,
		},
	}

	if err := clnt.Get(ctx, client.ObjectKeyFromObject(profile), profile); err != nil {
		return nil, err
	}

	if !profile.Data.Pinned {
		return nil, dwsv1alpha2.NewResourceError("expected a pinned container profile '%s', but found one that is not pinned", indexedResourceName(workflow, index)).WithFatal()
	}

	return profile, nil
}

func findContainerProfile(ctx context.Context, clnt client.Client, workflow *dwsv1alpha2.Workflow, index int) (*nnfv1alpha1.NnfContainerProfile, error) {
	args, err := dwdparse.BuildArgsMap(workflow.Spec.DWDirectives[index])
	if err != nil {
		return nil, err
	}

	name, found := args["profile"]
	if !found {
		return nil, fmt.Errorf("container directive '%s' has no profile key", workflow.Spec.DWDirectives[index])
	}

	profile := &nnfv1alpha1.NnfContainerProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: os.Getenv("NNF_CONTAINER_PROFILE_NAMESPACE"),
		},
	}

	if err := clnt.Get(ctx, client.ObjectKeyFromObject(profile), profile); err != nil {
		return nil, err
	}

	if profile.Data.Pinned {
		return nil, dwsv1alpha2.NewResourceError("expected container profile that is not pinned '%s', but found one that is pinned", indexedResourceName(workflow, index)).WithFatal()
	}

	// Determine whether the profile is restricted to a UserID/GroupID.
	restrictedMsg := "container profile '%s' is restricted to %s %d"
	if profile.Data.UserID != nil && *profile.Data.UserID != workflow.Spec.UserID {
		return nil, dwsv1alpha2.NewResourceError("").WithUserMessage(restrictedMsg, profile.Name, "UserID", *profile.Data.UserID).WithUser().WithFatal()
	}
	if profile.Data.GroupID != nil && *profile.Data.GroupID != workflow.Spec.GroupID {
		return nil, dwsv1alpha2.NewResourceError("").WithUserMessage(restrictedMsg, profile.Name, "GroupID", *profile.Data.GroupID).WithUser().WithFatal()

	}

	return profile, nil
}

func createPinnedContainerProfileIfNecessary(ctx context.Context, clnt client.Client, scheme *kruntime.Scheme, workflow *dwsv1alpha2.Workflow, index int, log logr.Logger) error {
	profile, err := findPinnedContainerProfile(ctx, clnt, workflow, index)
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	if profile != nil {
		return nil
	}

	profile, err = findContainerProfile(ctx, clnt, workflow, index)
	if err != nil {
		return err
	}

	pinnedProfile := &nnfv1alpha1.NnfContainerProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      indexedResourceName(workflow, index),
			Namespace: workflow.Namespace,
		},
	}

	profile.Data.DeepCopyInto(&pinnedProfile.Data)

	pinnedProfile.Data.Pinned = true

	dwsv1alpha2.AddOwnerLabels(pinnedProfile, workflow)

	if err := controllerutil.SetControllerReference(workflow, pinnedProfile, scheme); err != nil {
		log.Error(err, "failed to set controller reference on profile", "profile", pinnedProfile)
		return fmt.Errorf("failed to set controller reference on profile %s", client.ObjectKeyFromObject(pinnedProfile))
	}

	if err := clnt.Create(ctx, pinnedProfile); err != nil {
		return err
	}
	log.Info("Created pinned container profile", "resource", client.ObjectKeyFromObject(pinnedProfile))

	return nil
}
