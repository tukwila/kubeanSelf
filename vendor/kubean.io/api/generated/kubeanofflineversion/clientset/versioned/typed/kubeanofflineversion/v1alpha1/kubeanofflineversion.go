// Code generated by client-gen. DO NOT EDIT.

package v1alpha1

import (
	"context"
	"time"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
	v1alpha1 "kubean.io/api/apis/kubeanofflineversion/v1alpha1"
	scheme "kubean.io/api/generated/kubeanofflineversion/clientset/versioned/scheme"
)

// KuBeanOfflineVersionsGetter has a method to return a KuBeanOfflineVersionInterface.
// A group's client should implement this interface.
type KuBeanOfflineVersionsGetter interface {
	KuBeanOfflineVersions() KuBeanOfflineVersionInterface
}

// KuBeanOfflineVersionInterface has methods to work with KuBeanOfflineVersion resources.
type KuBeanOfflineVersionInterface interface {
	Create(ctx context.Context, kuBeanOfflineVersion *v1alpha1.KuBeanOfflineVersion, opts v1.CreateOptions) (*v1alpha1.KuBeanOfflineVersion, error)
	Update(ctx context.Context, kuBeanOfflineVersion *v1alpha1.KuBeanOfflineVersion, opts v1.UpdateOptions) (*v1alpha1.KuBeanOfflineVersion, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*v1alpha1.KuBeanOfflineVersion, error)
	List(ctx context.Context, opts v1.ListOptions) (*v1alpha1.KuBeanOfflineVersionList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.KuBeanOfflineVersion, err error)
	KuBeanOfflineVersionExpansion
}

// kuBeanOfflineVersions implements KuBeanOfflineVersionInterface
type kuBeanOfflineVersions struct {
	client rest.Interface
}

// newKuBeanOfflineVersions returns a KuBeanOfflineVersions
func newKuBeanOfflineVersions(c *KubeanV1alpha1Client) *kuBeanOfflineVersions {
	return &kuBeanOfflineVersions{
		client: c.RESTClient(),
	}
}

// Get takes name of the kuBeanOfflineVersion, and returns the corresponding kuBeanOfflineVersion object, and an error if there is any.
func (c *kuBeanOfflineVersions) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.KuBeanOfflineVersion, err error) {
	result = &v1alpha1.KuBeanOfflineVersion{}
	err = c.client.Get().
		Resource("kubeanofflineversions").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of KuBeanOfflineVersions that match those selectors.
func (c *kuBeanOfflineVersions) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.KuBeanOfflineVersionList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1alpha1.KuBeanOfflineVersionList{}
	err = c.client.Get().
		Resource("kubeanofflineversions").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested kuBeanOfflineVersions.
func (c *kuBeanOfflineVersions) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Resource("kubeanofflineversions").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a kuBeanOfflineVersion and creates it.  Returns the server's representation of the kuBeanOfflineVersion, and an error, if there is any.
func (c *kuBeanOfflineVersions) Create(ctx context.Context, kuBeanOfflineVersion *v1alpha1.KuBeanOfflineVersion, opts v1.CreateOptions) (result *v1alpha1.KuBeanOfflineVersion, err error) {
	result = &v1alpha1.KuBeanOfflineVersion{}
	err = c.client.Post().
		Resource("kubeanofflineversions").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(kuBeanOfflineVersion).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a kuBeanOfflineVersion and updates it. Returns the server's representation of the kuBeanOfflineVersion, and an error, if there is any.
func (c *kuBeanOfflineVersions) Update(ctx context.Context, kuBeanOfflineVersion *v1alpha1.KuBeanOfflineVersion, opts v1.UpdateOptions) (result *v1alpha1.KuBeanOfflineVersion, err error) {
	result = &v1alpha1.KuBeanOfflineVersion{}
	err = c.client.Put().
		Resource("kubeanofflineversions").
		Name(kuBeanOfflineVersion.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(kuBeanOfflineVersion).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the kuBeanOfflineVersion and deletes it. Returns an error if one occurs.
func (c *kuBeanOfflineVersions) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	return c.client.Delete().
		Resource("kubeanofflineversions").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *kuBeanOfflineVersions) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Resource("kubeanofflineversions").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched kuBeanOfflineVersion.
func (c *kuBeanOfflineVersions) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.KuBeanOfflineVersion, err error) {
	result = &v1alpha1.KuBeanOfflineVersion{}
	err = c.client.Patch(pt).
		Resource("kubeanofflineversions").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
