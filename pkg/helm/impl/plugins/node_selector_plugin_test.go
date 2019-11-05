package plugins

import (
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"testing"
)

func Test_NodeAffinityTransform(t *testing.T) {
	tests := []struct {
		unstructuredObj *unstructured.Unstructured
		nodeAffinity    NodeAffinity
		tolerations     []Toleration
		result          *unstructured.Unstructured
		err             error
	}{
		{
			unstructuredObj: convertObjToUnstructured(&appsv1.StatefulSet{
				Spec: appsv1.StatefulSetSpec{
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{
								{
									Name: "test-container",
								},
							},
						},
					},
				},
			}),
			nodeAffinity: NodeAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: NodeSelector{
					NodeSelectorTerms: []NodeSelectorTerm{
						{
							MatchExpressions: []NodeSelectorRequirement{
								{
									Key:      "aaa",
									Operator: v1.NodeSelectorOpDoesNotExist,
								},
							},
						},
						{
							MatchExpressions: []NodeSelectorRequirement{
								{
									Key:      "bbb",
									Operator: v1.NodeSelectorOpExists,
								},
							},
						},
					},
				},
			},
			result: convertObjToUnstructured(&appsv1.StatefulSet{
				Spec: appsv1.StatefulSetSpec{
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{
								{
									Name: "test-container",
								},
							},
							Affinity: &v1.Affinity{
								NodeAffinity: &v1.NodeAffinity{
									RequiredDuringSchedulingIgnoredDuringExecution: &v1.NodeSelector{
										NodeSelectorTerms: []v1.NodeSelectorTerm{
											{
												MatchExpressions: []v1.NodeSelectorRequirement{
													{
														Key:      "aaa",
														Operator: v1.NodeSelectorOpDoesNotExist,
													},
												},
											},
											{
												MatchExpressions: []v1.NodeSelectorRequirement{
													{
														Key:      "bbb",
														Operator: v1.NodeSelectorOpExists,
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			}),
			err: nil,
		},
		{
			unstructuredObj: convertObjToUnstructured(&appsv1.StatefulSet{
				Spec: appsv1.StatefulSetSpec{
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{
								{
									Name: "test-container",
								},
							},
							Affinity: &v1.Affinity{
								NodeAffinity: &v1.NodeAffinity{
									RequiredDuringSchedulingIgnoredDuringExecution: &v1.NodeSelector{
										NodeSelectorTerms: []v1.NodeSelectorTerm{
											{
												MatchExpressions: []v1.NodeSelectorRequirement{
													{
														Key:      "ccc",
														Operator: v1.NodeSelectorOpDoesNotExist,
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			}),
			nodeAffinity: NodeAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: NodeSelector{
					NodeSelectorTerms: []NodeSelectorTerm{
						{
							MatchExpressions: []NodeSelectorRequirement{
								{
									Key:      "aaa",
									Operator: v1.NodeSelectorOpDoesNotExist,
								},
							},
						},
						{
							MatchExpressions: []NodeSelectorRequirement{
								{
									Key:      "bbb",
									Operator: v1.NodeSelectorOpExists,
								},
							},
						},
					},
				},
			},
			result: convertObjToUnstructured(&appsv1.StatefulSet{
				Spec: appsv1.StatefulSetSpec{
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{
								{
									Name: "test-container",
								},
							},
							Affinity: &v1.Affinity{
								NodeAffinity: &v1.NodeAffinity{
									RequiredDuringSchedulingIgnoredDuringExecution: &v1.NodeSelector{
										NodeSelectorTerms: []v1.NodeSelectorTerm{
											{
												MatchExpressions: []v1.NodeSelectorRequirement{
													{
														Key:      "ccc",
														Operator: v1.NodeSelectorOpDoesNotExist,
													},
												},
											},
											{
												MatchExpressions: []v1.NodeSelectorRequirement{
													{
														Key:      "aaa",
														Operator: v1.NodeSelectorOpDoesNotExist,
													},
												},
											},
											{
												MatchExpressions: []v1.NodeSelectorRequirement{
													{
														Key:      "bbb",
														Operator: v1.NodeSelectorOpExists,
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			}),
			err: nil,
		},
	}

	for _, test := range tests {
		preferedAffinity, requredAffinity := convertToNodeAffinity(test.nodeAffinity)
		err := mergeNodeAffinity(test.unstructuredObj, preferedAffinity, requredAffinity)
		assert.IsType(t, test.err, err)
		assert.Equal(t, test.result, test.unstructuredObj)
	}
}

func Test_TolerationsTransform(t *testing.T) {
	tests := []struct {
		unstructuredObj *unstructured.Unstructured
		tolerations     []Toleration
		result          *unstructured.Unstructured
		err             error
	}{
		{
			unstructuredObj: convertObjToUnstructured(&appsv1.StatefulSet{
				Spec: appsv1.StatefulSetSpec{
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{
								{
									Name: "test-container",
								},
							},
						},
					},
				},
			}),
			tolerations: []Toleration{
				{
					Key:      "t1",
					Operator: v1.TolerationOpExists,
				},
				{
					Key:      "t2",
					Operator: v1.TolerationOpEqual,
					Value:    "v2",
				},
			},
			result: convertObjToUnstructured(&appsv1.StatefulSet{
				Spec: appsv1.StatefulSetSpec{
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{
								{
									Name: "test-container",
								},
							},
							Tolerations: []v1.Toleration{
								{
									Key:      "t1",
									Operator: v1.TolerationOpExists,
								},
								{
									Key:      "t2",
									Operator: v1.TolerationOpEqual,
									Value:    "v2",
								},
							},
						},
					},
				},
			}),
			err: nil,
		},
	}

	for _, test := range tests {
		tolerations := convertToToleration(test.tolerations)
		err := mergeNodeToleration(test.unstructuredObj, tolerations)
		assert.IsType(t, test.err, err)
		assert.Equal(t, test.result, test.unstructuredObj)
	}
}
