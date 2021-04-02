package template

import (
	"bytes"
	"fmt"
	"github.com/gogf/gf/database/gdb"
	"github.com/gogf/gf/frame/g"
	"github.com/gogf/gf/os/gfile"
	"github.com/gogf/gf/os/gtime"
	"github.com/gogf/gf/text/gregex"
	"github.com/gogf/gf/text/gstr"
	"github.com/olekukonko/tablewriter"
	"log"
	"strings"
	"testing"
)

func TestGenCode(t *testing.T) {
	tables := "t_sys_shop_merchant_record"
	project := "shop_manage"
	module := "shop_manage"
	if tables == "" || module == "" {
		return
	}
	tableNames := strings.Split(tables, ",")
	GenModule(module, project)
	for _, table := range tableNames {
		GenCode(project, table, module)
	}

}

func GenModule(module string, project string) {
	v := g.View()
	c := g.Config()
	folderPath := "./result/" + module
	moduleContent, _ := v.Parse("module.go.html", map[string]interface{}{
		"module":   module,
		"project":  project,
		"DateTime": gtime.Date(),
		"Author":   c.GetString("author"),
		"Email":    c.GetString("email"),
	})
	path := folderPath + gfile.Separator + "module.go"
	if err := gfile.PutContents(path, strings.TrimSpace(moduleContent)); err != nil {
		log.Fatalf("writing model content to '%s' failed: %v", path, err)
	}
}

func GenCode(project string, table string, module string) {
	//pathName := strings.ToLower(gstr.CamelCase(table))
	GenGo(project, table, module)
}

func GenGo(project string, table string, module string) {
	db := g.DB("default")
	folderPath := "result/" + module
	//generateModelContentFile(project, db, table, folderPath, table, "default")
	generateGormContentFile(project, db, table, folderPath, table, "default")
	generateControllerContentFile(project, db, table, folderPath, table, module, "default")
}

func generateControllerContentFile(project string, db gdb.DB, table string, folderPath, packageName, module, groupName string) {
	camelName := gstr.CamelCase(table)
	v := g.View()
	v.BindFunc("CameTable", CameTable)
	c := g.Config()
	tablePath := strings.ToLower(gstr.Replace(table, "_", "/"))
	controllerContent, err := v.Parse("controller.go.html", map[string]interface{}{
		"TplTableName":   table,
		"TplProject":     project,
		"TplModelName":   camelName,
		"TplGroupName":   groupName,
		"TplPackageName": packageName,
		"pathName":       tablePath,
		"moduleName":     module,
		"table":          table,
		"DateTime":       gtime.Date(),
		"Author":         c.GetString("author"),
		"Email":          c.GetString("email"),
	})
	if err != nil {
		print(err.Error())
	}
	name := gstr.Trim(gstr.SnakeCase(table), "-_.")
	path := folderPath + gfile.Separator + "controller" + gfile.Separator + name + "_controller.go"
	if err := gfile.PutContents(path, strings.TrimSpace(controllerContent)); err != nil {
		log.Fatalf("writing controller content to '%s' failed: %v", path, err)
	}
}
func generateGormContentFile(project string, db gdb.DB, table string, folderPath, packageName, groupName string) {
	fields, err := db.TableFields(table)
	if err != nil {
		log.Fatalf("fetching tables fields failed for table '%s':\n%v", table, err)
	}
	camelName := gstr.CamelCase(table)
	structDefine := generateStructDefinition(table, fields)
	columnDefine := ""
	index := 1
	for _, field := range fields {
		if index == len(fields) {
			columnDefine = columnDefine + "t." + field.Name + " as " + gstr.CamelCase(field.Name)
		} else {
			columnDefine = columnDefine + "t." + field.Name + " as " + gstr.CamelCase(field.Name) + ","
		}
		index = index + 1
	}
	v := g.View()
	c := g.Config()

	modelContent, err := v.Parse("gorm.go.html", map[string]interface{}{
		"TplTableName":    table,
		"TplProject":      project,
		"TplModelName":    camelName,
		"TplGroupName":    groupName,
		"columnDefine":    columnDefine,
		"TplPackageName":  packageName,
		"TplStructDefine": structDefine,
		"DateTime":        gtime.Date(),
		"Author":          c.GetString("author"),
		"Email":           c.GetString("email"),
	})
	name := gstr.Trim(gstr.SnakeCase(table), "-_.")
	if len(name) > 5 && name[len(name)-5:] == "_test" {
		// Add suffix to avoid the table name which contains "_test",
		// which would make the go file a testing file.
		name += "_table"
	}
	path := folderPath + gfile.Separator + "gormModel" + gfile.Separator + name + "_model.go"
	if err := gfile.PutContents(path, strings.TrimSpace(modelContent)); err != nil {
		log.Fatalf("writing model content to '%s' failed: %v", path, err)
	}
}
func generateStructDefinition(table string, fields map[string]*gdb.TableField) string {
	buffer := bytes.NewBuffer(nil)
	array := make([][]string, len(fields))
	for _, field := range fields {
		array[field.Index] = generateStructField(field)
	}
	tw := tablewriter.NewWriter(buffer)
	tw.SetBorder(false)
	tw.SetRowLine(false)
	tw.SetAutoWrapText(false)
	tw.SetColumnSeparator("")
	tw.AppendBulk(array)
	tw.Render()
	stContent := buffer.String()
	// Let's do this hack for tablewriter!
	stContent = gstr.Replace(stContent, "  #", "")
	buffer.Reset()
	buffer.WriteString("type " + gstr.CamelCase(table) + " struct {\n")
	buffer.WriteString(stContent)
	buffer.WriteString("}")
	return buffer.String()
}
func generateStructField(field *gdb.TableField) []string {
	var typeName, ormTag, jsonTag string
	t, _ := gregex.ReplaceString(`\(.+\)`, "", field.Type)
	arr := gstr.Split(t, " ")
	t = strings.ToLower(arr[0])
	t = strings.ToLower(t)
	fmt.Printf("当前Type:%s", field.Type)
	fmt.Printf("当前字段:%s:%s\n", field.Name, t)
	switch t {
	case "binary", "varbinary", "blob", "tinyblob", "mediumblob", "longblob":
		typeName = "[]byte"

	case "bit", "int", "tinyint", "smallint", "mediumint":
		if gstr.ContainsI(field.Type, "unsigned") {
			typeName = "uint"
		} else {
			typeName = "int"
		}

	case "bigint":
		if gstr.ContainsI(field.Type, "unsigned") {
			typeName = "uint64"
		} else {
			typeName = "int64"
		}

	case "float", "double", "decimal":
		typeName = "float64"

	case "bool":
		typeName = "bool"

	case "datetime", "timestamp", "date", "time":
		typeName = "*time.Time"

	default:
		// Auto detecting type.
		switch {
		case strings.Contains(t, "int64"):
			typeName = "int64"
		case strings.Contains(t, "int"):
			typeName = "int"
		case strings.Contains(t, "text") || strings.Contains(t, "char"):
			typeName = "string"
		case strings.Contains(t, "float") || strings.Contains(t, "double"):
			typeName = "float64"
		case strings.Contains(t, "bool"):
			typeName = "bool"
		case strings.Contains(t, "binary") || strings.Contains(t, "blob"):
			typeName = "[]byte"
		case strings.Contains(t, "date") || strings.Contains(t, "time"):
			typeName = "*time.Time"
		default:
			typeName = "string"
		}
	}
	ormTag = field.Name
	jsonTag = gstr.CamelCase(field.Name)
	if gstr.ContainsI(field.Key, "pri") {
		ormTag += ",primary"
	}
	if gstr.ContainsI(field.Key, "uni") {
		ormTag += ",unique"
	}
	return []string{
		"    #" + gstr.CamelCase(field.Name),
		" #" + typeName,
		" #" + fmt.Sprintf("`"+`gorm:"%s"`, ormTag),
		//" #" + fmt.Sprintf(`json:"%s,omitempty" gconv:"%s,omitempty"`+"`", gstr.CamelCase(field.Name), jsonTag),
		" #" + fmt.Sprintf(`json:"%s,omitempty"`+"`", jsonTag),
	}
}

func CamelCase(name string) string {
	return gstr.CamelCase(name)
}
func ScamelCase(name string) string {
	return gstr.CamelLowerCase(name)
}
func CameTable(name string) string {
	return strings.ToLower(gstr.Replace(name, "_", "/"))
}

func SubTable(name string) string {
	tables := gstr.Split(name, "_")
	println(tables)
	return tables[1]
}
