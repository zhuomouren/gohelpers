package gotemplate

import (
	"html/template"
	"path/filepath"
	"regexp"
	"strings"
)

type goTemplate struct {
	path string
	data string
}

func newTmpl(path, data string) *goTemplate {
	return &goTemplate{
		path: path,
		data: data,
	}
}

type GoTemplate struct {
	// Root      string
	templates map[string][]*goTemplate
	exts      []string
}

var TemplateHelper = GoTemplate{}

func NewTemplate() {

}

func (this *GoTemplate) AddTemplate(root, path, data string) {
	gotmpl := newTmpl(path, data)

	if tmpls, ok := this.templates[root]; ok {
		has := false
		for i, tmpl := range tmpls {
			if strings.EqualFold(tmpl.path, gotmpl.path) {
				has = true
				tmpls[i].data = gotmpl.data
			}
		}

		if !has {
			tmpls = append(tmpls, gotmpl)
		}

		this.templates[root] = tmpls
	} else {
		this.templates[root] = append(this.templates[root], gotmpl)
	}
}

func (this *GoTemplate) AddExt(exts ...string) {
	this.exts = append(this.exts, exts...)
}

func (this *GoTemplate) Parse(paths ...string) *template.Template {
	t := template.New("")

	return t
}

func (this *GoTemplate) parse() *template.Template {
	t := template.New("")
	var err error
	for root, tmpls := range this.templates {
		for _, tmpl := range tmpls {
			name := filepath.ToSlash(filepath.Clean(tmpl.path))
			data := this.normalizedDefine(root, tmpl.path, tmpl.data)
			data = this.normalizedTemplate(root, tmpl.path, data)
			t, err = t.New(name).Parse(data)
			if err != nil {
				panic("template parse file err:" + err.Error())
			}
		}
	}
	return t
}

func (this *GoTemplate) isExist(filename string) bool {
	for _, tmpls := range this.templates {
		for _, tmpl := range tmpls {
			if strings.EqualFold(filename, tmpl.path) {
				return true
			}
		}
	}
	return false
}

func (this *GoTemplate) normalizedDefine(root, path, data string) string {
	templatePrefix := "{{"
	definePairs := make(map[string]string, 0)
	reg := regexp.MustCompile(templatePrefix + "[ ]*define[ ]+\"([^\"]+)\"")
	allSub := reg.FindAllStringSubmatch(string(data), -1)
	for _, sub := range allSub {
		if len(sub) == 2 {
			if _, ok := definePairs[sub[1]]; ok {
				continue
			}
			// fmt.Println("define: ", sub[0], sub[1])
			tplFile := filepath.Clean(filepath.Join(root, sub[1]))
			if this.isExist(tplFile) {
				definePairs[sub[1]] = tplFile
			}
		}
	}

	for key, val := range definePairs {
		data = strings.Replace(data, key, val, -1)
	}

	return data
}

func (this *GoTemplate) normalizedTemplate(root, path, data string) string {
	templatePrefix := "{{"
	templatePairs := make(map[string]string, 0)
	reg := regexp.MustCompile(templatePrefix + "[ ]*template[ ]+\"([^\"]+)\"")
	allSub := reg.FindAllStringSubmatch(string(data), -1)
	for _, sub := range allSub {
		if len(sub) == 2 {
			if _, ok := templatePairs[sub[1]]; ok {
				continue
			}

			tplFile := filepath.Clean(filepath.Join(root, sub[1]))
			if this.isExist(tplFile) {
				templatePairs[sub[1]] = tplFile
			}
		}
	}

	for key, val := range templatePairs {
		data = strings.Replace(data, key, val, -1)
	}

	return data
}
