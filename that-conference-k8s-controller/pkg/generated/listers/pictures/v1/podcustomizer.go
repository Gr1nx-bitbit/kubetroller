/*
Copyright The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by lister-gen. DO NOT EDIT.

package v1

import (
	v1 "github.com/eddiezane/that-conference-k8s-controller/pkg/apis/pictures/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// PodCustomizerLister helps list PodCustomizers.
// All objects returned here must be treated as read-only.
type PodCustomizerLister interface {
	// List lists all PodCustomizers in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1.PodCustomizer, err error)
	// PodCustomizers returns an object that can list and get PodCustomizers.
	PodCustomizers(namespace string) PodCustomizerNamespaceLister
	PodCustomizerListerExpansion
}

// podCustomizerLister implements the PodCustomizerLister interface.
type podCustomizerLister struct {
	indexer cache.Indexer
}

// NewPodCustomizerLister returns a new PodCustomizerLister.
func NewPodCustomizerLister(indexer cache.Indexer) PodCustomizerLister {
	return &podCustomizerLister{indexer: indexer}
}

// List lists all PodCustomizers in the indexer.
func (s *podCustomizerLister) List(selector labels.Selector) (ret []*v1.PodCustomizer, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1.PodCustomizer))
	})
	return ret, err
}

// PodCustomizers returns an object that can list and get PodCustomizers.
func (s *podCustomizerLister) PodCustomizers(namespace string) PodCustomizerNamespaceLister {
	return podCustomizerNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// PodCustomizerNamespaceLister helps list and get PodCustomizers.
// All objects returned here must be treated as read-only.
type PodCustomizerNamespaceLister interface {
	// List lists all PodCustomizers in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1.PodCustomizer, err error)
	// Get retrieves the PodCustomizer from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v1.PodCustomizer, error)
	PodCustomizerNamespaceListerExpansion
}

// podCustomizerNamespaceLister implements the PodCustomizerNamespaceLister
// interface.
type podCustomizerNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all PodCustomizers in the indexer for a given namespace.
func (s podCustomizerNamespaceLister) List(selector labels.Selector) (ret []*v1.PodCustomizer, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1.PodCustomizer))
	})
	return ret, err
}

// Get retrieves the PodCustomizer from the indexer for a given namespace and name.
func (s podCustomizerNamespaceLister) Get(name string) (*v1.PodCustomizer, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1.Resource("podcustomizer"), name)
	}
	return obj.(*v1.PodCustomizer), nil
}
