package transwarpjsonnet

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"path"
	"path/filepath"
	"strconv"
	"text/template"

	"github.com/Masterminds/sprig"
	"WarpCloud/walm/pkg/util/transwarpjsonnet/memkv"
)

type confdVolParam struct {
	ConfigMapKey     string `json:"configMapKey"`
	FileLocation     string `json:"fileLocation"`
	VolumeMountPath  string `json:"volumeMountPath"`
	VolConfigMapMode int    `json:"volConfigMapMode"`
}

func renderConfdFiles(name string, confdKV interface{}, confdFilesConfig interface{}, mainPath string) (interface{}, error) {
	confdFileParamList := make([]confdVolParam, 0)

	confdFileConfigBytes, err := json.Marshal(confdFilesConfig)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(confdFileConfigBytes, &confdFileParamList)
	if err != nil {
		return nil, err
	}

	resObj := configFilesResult{
		Name:             name,
		Md5Checksum:      "",
		VolumeMountList:  make([]volumeMount, 0),
		VolumeList:       make([]volume, 0),
		ConfigMapDataMap: make(map[string]string, 0),
	}

	var allContent string
	for _, confdFileParam := range confdFileParamList {
		filePath := confdFileParam.FileLocation
		if mainPath != "" {
			filePath = filepath.Join(filepath.Dir(mainPath), filePath)
		}

		content, err := renderConfdFile(filePath, confdKV)
		if err != nil {
			return nil, err
		}
		allContent += content

		// 生成 ConfigMap
		resObj.ConfigMapDataMap[confdFileParam.ConfigMapKey] = content

		// 生成 Volumes Mounts
		volumeMountConfig := volumeMount{
			Name:      name,
			MountPath: confdFileParam.VolumeMountPath,
			SubPath:   confdFileParam.ConfigMapKey,
		}
		resObj.VolumeMountList = append(resObj.VolumeMountList, volumeMountConfig)

		// 生成 Volumes Config
		volumeConfig := volume{
			Key:  confdFileParam.ConfigMapKey,
			Path: confdFileParam.ConfigMapKey,
			Mode: 420,
		}
		if confdFileParam.VolConfigMapMode != 0 {
			volumeConfig.Mode = confdFileParam.VolConfigMapMode
		}
		resObj.VolumeList = append(resObj.VolumeList, volumeConfig)
	}
	md5Data := md5.Sum([]byte(allContent))
	resObj.Md5Checksum = string(md5Data[:])

	var res interface{}
	data, err := json.Marshal(resObj)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(data, &res)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func renderConfdFile(filePath string, confdkv interface{}) (string, error) {
	var t *template.Template
	store := memkv.New()
	vars := make(map[string]string)
	yamlMap := make(map[string]interface{})

	byteKV, _ := json.Marshal(confdkv)
	err := json.Unmarshal(byteKV, &yamlMap)

	err = nodeWalk(yamlMap, "", vars)
	if err != nil {
		return "", err
	}

	for k, v := range vars {
		store.Set(path.Join("/", k), v)
	}

	t, err = template.New(filepath.Base(filePath)).
		Funcs(sprig.TxtFuncMap()).
		Funcs(store.FuncMap).ParseFiles(filePath)
	if err != nil {
		return "", err
	}

	var fileTpl bytes.Buffer
	err = t.Execute(&fileTpl, confdkv)
	if err != nil {
		return "", err
	}
	return fileTpl.String(), nil
}

// nodeWalk recursively descends nodes, updating vars.
func nodeWalk(node interface{}, key string, vars map[string]string) error {
	switch node.(type) {
	case []interface{}:
		for i, j := range node.([]interface{}) {
			key := path.Join(key, strconv.Itoa(i))
			nodeWalk(j, key, vars)
		}
	case map[string]interface{}:
		for k, v := range node.(map[string]interface{}) {
			key := path.Join(key, k)
			nodeWalk(v, key, vars)
		}
	case string:
		vars[key] = node.(string)
	case int:
		vars[key] = strconv.Itoa(node.(int))
	case bool:
		vars[key] = strconv.FormatBool(node.(bool))
	case float64:
		vars[key] = strconv.FormatFloat(node.(float64), 'f', -1, 64)
	}
	return nil
}
