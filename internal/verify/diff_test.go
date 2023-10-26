// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package verify

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	"github.com/hashicorp/terraform-provider-aws/internal/flex"
	tftags "github.com/hashicorp/terraform-provider-aws/internal/tags"
)

func TestSuppressEquivalentRoundedTime(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		old        string
		new        string
		layout     string
		d          time.Duration
		equivalent bool
	}{
		{
			old:        "2024-04-19T23:00:00.000Z",
			new:        "2024-04-19T23:00:13.000Z",
			layout:     time.RFC3339,
			d:          time.Minute,
			equivalent: true,
		},
		{
			old:        "2024-04-19T23:01:00.000Z",
			new:        "2024-04-19T23:00:45.000Z",
			layout:     time.RFC3339,
			d:          time.Minute,
			equivalent: true,
		},
		{
			old:        "2024-04-19T23:00:00.000Z",
			new:        "2024-04-19T23:00:45.000Z",
			layout:     time.RFC3339,
			d:          time.Minute,
			equivalent: false,
		},
		{
			old:        "2024-04-19T23:00:00.000Z",
			new:        "2024-04-19T23:00:45.000Z",
			layout:     time.RFC3339,
			d:          time.Hour,
			equivalent: true,
		},
	}

	for i, tc := range testCases {
		value := SuppressEquivalentRoundedTime(tc.layout, tc.d)("test_property", tc.old, tc.new, nil)

		if tc.equivalent && !value {
			t.Fatalf("expected test case %d to be equivalent", i)
		}

		if !tc.equivalent && value {
			t.Fatalf("expected test case %d to not be equivalent", i)
		}
	}
}

func TestDiffStringMaps(t *testing.T) {
	t.Parallel()

	cases := []struct {
		Old, New                  map[string]interface{}
		Create, Remove, Unchanged map[string]interface{}
	}{
		// Add
		{
			Old: map[string]interface{}{
				"foo": "bar",
			},
			New: map[string]interface{}{
				"foo": "bar",
				"bar": "baz",
			},
			Create: map[string]interface{}{
				"bar": "baz",
			},
			Remove: map[string]interface{}{},
			Unchanged: map[string]interface{}{
				"foo": "bar",
			},
		},

		// Modify
		{
			Old: map[string]interface{}{
				"foo": "bar",
			},
			New: map[string]interface{}{
				"foo": "baz",
			},
			Create: map[string]interface{}{
				"foo": "baz",
			},
			Remove: map[string]interface{}{
				"foo": "bar",
			},
			Unchanged: map[string]interface{}{},
		},

		// Overlap
		{
			Old: map[string]interface{}{
				"foo":   "bar",
				"hello": "world",
			},
			New: map[string]interface{}{
				"foo":   "baz",
				"hello": "world",
			},
			Create: map[string]interface{}{
				"foo": "baz",
			},
			Remove: map[string]interface{}{
				"foo": "bar",
			},
			Unchanged: map[string]interface{}{
				"hello": "world",
			},
		},

		// Remove
		{
			Old: map[string]interface{}{
				"foo": "bar",
				"bar": "baz",
			},
			New: map[string]interface{}{
				"foo": "bar",
			},
			Create: map[string]interface{}{},
			Remove: map[string]interface{}{
				"bar": "baz",
			},
			Unchanged: map[string]interface{}{
				"foo": "bar",
			},
		},
	}

	for i, tc := range cases {
		c, r, u := DiffStringMaps(tc.Old, tc.New)
		cm := flex.PointersMapToStringList(c)
		rm := flex.PointersMapToStringList(r)
		um := flex.PointersMapToStringList(u)
		if !reflect.DeepEqual(cm, tc.Create) {
			t.Fatalf("%d: bad create: %#v", i, cm)
		}
		if !reflect.DeepEqual(rm, tc.Remove) {
			t.Fatalf("%d: bad remove: %#v", i, rm)
		}
		if !reflect.DeepEqual(um, tc.Unchanged) {
			t.Fatalf("%d: bad unchanged: %#v", i, rm)
		}
	}
}

func TestSetTagsDiff(t *testing.T) {

	type testCase struct {
		name              string
		state             cty.Value
		config            cty.Value
		defaultTagsConfig *tftags.DefaultConfig
		ignoreTagsConfig  *tftags.IgnoreConfig
		expectedTagsAll   cty.Value
		expectedNoTagsAll bool
	}

	testCases := []testCase{
		{
			name: "creating with no tags",
			state: cty.ObjectVal(map[string]cty.Value{
				"tags": cty.MapValEmpty(cty.String),
			}),
			config: cty.ObjectVal(map[string]cty.Value{
				"tags": cty.MapValEmpty(cty.String),
			}),
			// This behavior is strange, why is this the answer unknown instead of not
			// setting the tags_all at all?
			expectedTagsAll: cty.UnknownVal(cty.Map(cty.String)),
		},
		{
			name:  "basic tags get copied to tags_all",
			state: cty.ObjectVal(map[string]cty.Value{}),
			config: cty.ObjectVal(map[string]cty.Value{
				"tags": cty.MapVal(map[string]cty.Value{
					"tag1": cty.StringVal("tag1v"),
				}),
			}),
			expectedTagsAll: cty.MapVal(map[string]cty.Value{
				"tag1": cty.StringVal("tag1v"),
			}),
		},
		{
			name: "basic tags not changing get copied to tags_all",
			state: cty.ObjectVal(map[string]cty.Value{
				"tags": cty.MapVal(map[string]cty.Value{
					"tag1": cty.StringVal("tag1v"),
				}),
			}),
			config: cty.ObjectVal(map[string]cty.Value{
				"tags": cty.MapVal(map[string]cty.Value{
					"tag1": cty.StringVal("tag1v"),
				}),
			}),
			expectedTagsAll: cty.MapVal(map[string]cty.Value{
				"tag1": cty.StringVal("tag1v"),
			}),
		},
		{
			name:  "unknowns in tags plan cause unknown tags_all",
			state: cty.ObjectVal(map[string]cty.Value{}),
			config: cty.ObjectVal(map[string]cty.Value{
				"tags": cty.UnknownVal(cty.Map(cty.String)),
			}),
			expectedTagsAll: cty.UnknownVal(cty.Map(cty.String)),
		},
		{
			name:  "setting a simple empty-value tag works as expected",
			state: cty.ObjectVal(map[string]cty.Value{}),
			config: cty.ObjectVal(map[string]cty.Value{
				"tags": cty.MapVal(map[string]cty.Value{
					"tag1": cty.StringVal(""),
				}),
			}),
			expectedTagsAll: cty.MapVal(map[string]cty.Value{
				// This is really not right. It looks like
				"tag1": cty.UnknownVal(cty.String),
			}),
		},
	}

	for _, tc := range testCases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			d, err := checkSetTagsDiff(tc.state, tc.config, tc.defaultTagsConfig, tc.ignoreTagsConfig)
			if err != nil {
				t.Error(err)
				t.FailNow()
			}

			_, actualGotTagsAll := d.GetAttribute("tags_all.%")
			if actualGotTagsAll != !tc.expectedNoTagsAll {
				t.Logf("A = %v", d.Attributes)
				for k, v := range d.Attributes {
					t.Logf("%v => %v Computed=%v", k, v, v.NewComputed)

				}
				t.Errorf("tags_all handled incorrectly\n  expected to be set: %v\n  was set: %v\n",
					!tc.expectedNoTagsAll,
					actualGotTagsAll,
				)
			}

			if tc.expectedNoTagsAll {
				return
			}

			applied, err := d.ApplyToValue(tc.state, resourceWithTags().CoreConfigSchema())
			if err != nil {
				t.Error(err)
				t.FailNow()
			}

			actual := applied.GetAttr("tags_all")
			expected := tc.expectedTagsAll

			if !reflect.DeepEqual(actual, expected) {
				t.Errorf("tags_all not set as expected\n  actual: %v\n  expected: %v\n",
					actual.GoString(),
					expected.GoString(),
				)
				t.FailNow()
			}
		})
	}
}

func resourceWithTags() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"tags":     tagsSchema(),
			"tags_all": tagsSchemaTrulyComputed(),
		},
	}
}

func checkSetTagsDiff(
	state cty.Value,
	config cty.Value,
	defaultTagsConfig *tftags.DefaultConfig,
	ignoreTagsConfig *tftags.IgnoreConfig,
) (*terraform.InstanceDiff, error) {
	ctx := context.Background()
	resource := resourceWithTags()
	instanceState := terraform.NewInstanceStateShimmedFromValue(state, 0)
	// Usually TF CLI sets RawPlan to a merge of state and config, but it does not matter for this test.
	instanceState.RawPlan = config
	resourceConfig := terraform.NewResourceConfigShimmed(config, resource.CoreConfigSchema())
	meta := &conns.AWSClient{}
	meta.DefaultTagsConfig = defaultTagsConfig
	meta.IgnoreTagsConfig = ignoreTagsConfig
	m := schema.InternalMap(resource.Schema)
	diff, err := m.Diff(ctx, instanceState, resourceConfig, SetTagsDiff, meta, false /* handleRequiresNew */)
	if err != nil {
		return nil, err
	}
	return diff, nil
}

func tagsSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeMap,
		Optional: true,
		Elem:     &schema.Schema{Type: schema.TypeString},
	}
}

func tagsSchemaTrulyComputed() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeMap,
		Computed: true,
		Elem:     &schema.Schema{Type: schema.TypeString},
	}
}
