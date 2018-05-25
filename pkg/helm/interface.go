package helm

import (
	"bytes"
	"errors"
	
	"io/ioutil"
	"strings"
	
	. "walm/pkg/util/log"

	"gopkg.in/pipe.v2"
)

type Interface struct {
	cmd  string
	path string
}

var Helm *Interface

func init() {
	Helm = &Interface{cmd: "helm "}
}

func (inst *Interface) makeCmd(subcmd string, args, flags []string) (error, string) {
	if len(args) == 0 && len(flags) == 0 {
		return errors.New("no args and no flags"), ""
	}
	cmd := inst.cmd
	cmd += subcmd
	cmd += strings.Join(flags, " ")
	cmd += strings.Join(args, " ")
	return nil, cmd
}

func execPipeLine(cmd string) (error, *bytes.Buffer) {
	Log.Debugf("beging to exec cmd:%s\n", cmd)
	b := &bytes.Buffer{}
	p := pipe.Line(
		pipe.Exec(cmd),
		pipe.Write(b),
	)
	err := pipe.Run(p)
	return err, b
}

// MakeValueFile creates a temporary file in TempDir (see os.TempDir)
// and writes values to the file and resturn its name. It is the caller's responsibility
// to remove the file returned if necessary.
func (inst *Interface) MakeValueFile(data []byte) (string, error) {
	tmpFile, err := ioutil.TempFile("", "tmp-values-")
	if err != nil {
		return "", err
	}
	defer tmpFile.Close()

	if _, err = tmpFile.Write(data); err != nil {
		return tmpFile.Name(), err
	}
	tmpFile.Sync()
	return tmpFile.Name(), nil
}

func (inst *Interface) Detele(args, flags []string) error {
	if err, cmd := inst.makeCmd("delete", args, flags); err != nil {
		return err
	} else {
		err, _ = execPipeLine(cmd)
		return err
	}
}
func (inst *Interface) Rollback(args, flags []string) error {
	if err, cmd := inst.makeCmd("rollback", args, flags); err != nil {
		return err
	} else {
		err, _ = execPipeLine(cmd)
		return err
	}
}

func (inst *Interface) UpdateRepo() error {
	if err, cmd := inst.makeCmd("repo", []string{"update"} , []string{}); err != nil {
		return err
	} else {
		err, _ = execPipeLine(cmd)
		return err
	}
}

func (inst *Interface) DeplyApplications(args, flags []string) error {
	//update repo before install chart
	if err := inst.UpdateRepo();err!=nil{
		return err
	}

	if err, cmd := inst.makeCmd("install", args, flags); err != nil {
		return err
	} else {
		err, _ = execPipeLine(cmd)
		return err
	}
}
func (inst *Interface) UpdateApplications(args, flags []string) error {
	if err, cmd := inst.makeCmd("upgrade", args, flags); err != nil {
		return err
	} else {
		err, _ = execPipeLine(cmd)
		return err
	}
}

func (inst *Interface) StatusApplications(args, flags []string) (string, error) {
	if err, cmd := inst.makeCmd("status", args, flags); err != nil {
		return "", err
	} else {
		var b *bytes.Buffer
		err, b = execPipeLine(cmd)
		return b.String(), err
	}
}

func (inst *Interface) ListApplications(args, flags []string) (*bytes.Buffer, error) {

	if err, cmd := inst.makeCmd("list", args, flags); err != nil {
		return &bytes.Buffer{}, err
	} else {
		var b *bytes.Buffer
		err, b = execPipeLine(cmd)
		return b, err
	}
}