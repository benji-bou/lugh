package core

import (
	"testing"

	"gopkg.in/yaml.v3"
)

var testYmlTpl = `name: test
description: test template
author: bbo
version: 0.1
pipeline:
  martianProxy:
    plugin: martianProxy
    config:
      scope: bioserenity.com
      otherConfig:
        - tata
        - titi
    childs:
      memfilter:
        plugin: memfilter
        pipe:
          - goTemplate:
              format: json
              pattern: "url: {{ .request.url }},\nmethod: {{.request.method}}\n\n"
          - regex:
              pattern: (?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z0-9][a-z0-9-]{0,61}[a-z0-9]
          - insert:
              content: "\n"
          - split:
              sep: "\n"
        childs:
          rawFile:
            plugin: rawfile
            pipe:
              - insert:
                  content: "\n"

`

func testPipe(t *testing.T, stageName string, pipes []map[string]yaml.Node, orderedPipes ...string) {
	if len(pipes) != len(orderedPipes) {
		t.Errorf("on stage %s, pipes count not correct, have: %d, expected: %d", stageName, len(pipes), len(orderedPipes))
	}
	for i, p := range orderedPipes {
		if countPipe := len(pipes[i]); countPipe != 1 {
			t.Errorf("stage %s should have a single pipe register at pos %d", stageName, i)

		}
		if _, exist := pipes[i][p]; !exist {
			t.Errorf(" stage %s should have a %s pipe at pos %d", stageName, p, i)
		}

	}
}

func testChilds(t *testing.T, stageName string, childs map[string]Pipeline, childsExpected ...string) {
	if len(childs) != len(childsExpected) {
		t.Errorf("on stage %s, childs count not correct, have: %d, expected: %d", stageName, len(childs), len(childsExpected))
	}
	for _, c := range childsExpected {
		if _, exist := childs[c]; !exist {
			t.Errorf(" stage %s should have a %s child", stageName, c)
		}

	}
}

func TestLoadTemplate(t *testing.T) {

	// rawInput := map[string]any{}
	// err := yaml.Unmarshal([]byte(testYmlTpl), &rawInput)
	// if err != nil {
	// 	t.Error(err)
	// }

	tpl := Template{}
	err := yaml.Unmarshal([]byte(testYmlTpl), &tpl)
	if err != nil {
		t.Error(err)
	}
	martian, exist := tpl.Pipeline["martianProxy"]
	if !exist {
		t.Error("root node invalid should be martianProxy")
	}
	testPipe(t, "martian", martian.Pipe)
	testChilds(t, "martian", martian.Childs, "memfilter")

	memfilter := martian.Childs["memfilter"]

	testPipe(t, "memfilter", memfilter.Pipe,
		"goTemplate",
		"regex",
		"insert",
		"split",
	)

	testChilds(t, "memfilter", memfilter.Childs, "rawFile")
	rawFile := memfilter.Childs["rawFile"]
	testChilds(t, "rawFile", rawFile.Childs)
	testPipe(t, "rawFile", rawFile.Pipe, "insert")
}

func testNodeChildsId(rootNode SecNode, expected ...string) {

}

func TestPluginNodeTreeLoading(t *testing.T) {

	tpl := Template{}
	err := yaml.Unmarshal([]byte(testYmlTpl), &tpl)
	if err != nil {
		t.Error(err)
	}
	secNode, err := tpl.GetSecNodeTree()
	if err != nil {
		t.Error(err)
	}
	if secNode.GetID() != "martianProxy" {
		t.Error("root node should be a martianProxy")
	}
	testNodeChildsId(secNode, "memfilter")
	martianChildNodes := secNode.GetChilds()
	memFilterNode := martianChildNodes["memfilter"]
	testNodeChildsId(memFilterNode, "rawFile")

}
