/*

Copyright 2019 Transwarp All rights reserved.
*/
// Code generated by lister-gen. DO NOT EDIT.

package v1alpha1

import (
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
	v1alpha1 "transwarp/isomateset-client/pkg/apis/apiextensions.transwarp.io/v1alpha1"
)

// IsomateSetLister helps list IsomateSets.
type IsomateSetLister interface {
	// List lists all IsomateSets in the indexer.
	List(selector labels.Selector) (ret []*v1alpha1.IsomateSet, err error)
	// IsomateSets returns an object that can list and get IsomateSets.
	IsomateSets(namespace string) IsomateSetNamespaceLister
	IsomateSetListerExpansion
}

// isomateSetLister implements the IsomateSetLister interface.
type isomateSetLister struct {
	indexer cache.Indexer
}

// NewIsomateSetLister returns a new IsomateSetLister.
func NewIsomateSetLister(indexer cache.Indexer) IsomateSetLister {
	return &isomateSetLister{indexer: indexer}
}

// List lists all IsomateSets in the indexer.
func (s *isomateSetLister) List(selector labels.Selector) (ret []*v1alpha1.IsomateSet, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.IsomateSet))
	})
	return ret, err
}

// IsomateSets returns an object that can list and get IsomateSets.
func (s *isomateSetLister) IsomateSets(namespace string) IsomateSetNamespaceLister {
	return isomateSetNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// IsomateSetNamespaceLister helps list and get IsomateSets.
type IsomateSetNamespaceLister interface {
	// List lists all IsomateSets in the indexer for a given namespace.
	List(selector labels.Selector) (ret []*v1alpha1.IsomateSet, err error)
	// Get retrieves the IsomateSet from the indexer for a given namespace and name.
	Get(name string) (*v1alpha1.IsomateSet, error)
	IsomateSetNamespaceListerExpansion
}

// isomateSetNamespaceLister implements the IsomateSetNamespaceLister
// interface.
type isomateSetNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all IsomateSets in the indexer for a given namespace.
func (s isomateSetNamespaceLister) List(selector labels.Selector) (ret []*v1alpha1.IsomateSet, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.IsomateSet))
	})
	return ret, err
}

// Get retrieves the IsomateSet from the indexer for a given namespace and name.
func (s isomateSetNamespaceLister) Get(name string) (*v1alpha1.IsomateSet, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1alpha1.Resource("isomateset"), name)
	}
	return obj.(*v1alpha1.IsomateSet), nil
}