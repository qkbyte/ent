// Copyright 2019-present Facebook Inc. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package printer

import (
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"

	"github.com/qkbyte/ent/entc/gen"

	"github.com/olekukonko/tablewriter"
)

// A Config controls the output of Fprint.
type Config struct {
	io.Writer
}

// Print prints a table description of the graph to the given writer.
func (p Config) Print(g *gen.Graph) {
	for _, n := range g.Nodes {
		p.node(n)
	}
}

// Print prints a table grpc message of the graph to the given writer.
func (p Config) Message(g *gen.Graph) {
	for _, n := range g.Nodes {
		p.grpc(n)
	}
}

// Fmessage executes "pretty-printer" on the given writer.
func Fmessage(w io.Writer, g *gen.Graph) {
	Config{Writer: w}.Message(g)
}

// Fprint executes "pretty-printer" on the given writer.
func Fprint(w io.Writer, g *gen.Graph) {
	Config{Writer: w}.Print(g)
}

// node returns description of a type. The format of the description is:
//
//	Type:
//			<Fields Table>
//
//			<Edges Table>
//
func (p Config) node(t *gen.Type) {
	var (
		b      strings.Builder
		table  = tablewriter.NewWriter(&b)
		header = []string{"Field", "Type", "FieldComment", "Unique", "Optional", "Nillable", "Default", "UpdateDefault", "Immutable", "StructTag", "Validators"}
	)
	b.WriteString(fmt.Sprintf("%s(%s):\n", t.Name, t.Comment()))
	table.SetAutoFormatHeaders(false)
	table.SetHeader(header)
	for _, f := range append([]*gen.Field{t.ID}, t.Fields...) {
		v := reflect.ValueOf(*f)
		row := make([]string, len(header))
		for i := range row {
			field := v.FieldByNameFunc(func(name string) bool {
				// The first field is mapped from "Name" to "Field".
				return name == "Name" && i == 0 || name == header[i]
			})
			row[i] = fmt.Sprint(field.Interface())
		}
		table.Append(row)
	}
	table.Render()
	table = tablewriter.NewWriter(&b)
	table.SetAutoFormatHeaders(false)
	table.SetHeader([]string{"Edge", "Type", "EdgeComment", "Inverse", "BackRef", "Relation", "Unique", "Optional"})
	for _, e := range t.Edges {
		table.Append([]string{
			e.Name,
			e.Type.Name,
			e.EdgeComment,
			strconv.FormatBool(e.IsInverse()),
			e.Inverse,
			e.Rel.Type.String(),
			strconv.FormatBool(e.Unique),
			strconv.FormatBool(e.Optional),
		})
	}
	if table.NumLines() > 0 {
		table.Render()
	}
	io.WriteString(p, strings.ReplaceAll(b.String(), "\n", "\n\t")+"\n")
}

// grpc returns grpc message of a type. The format of the message is:
//
//	Type:
//			<Fields Table>
//
//			<Edges Table>
//
func (p Config) grpc(t *gen.Type) {
	var b strings.Builder
	b.WriteString("//" + t.Comment() + "\n")
	b.WriteString("message " + t.Name + "{\n")
	var index = 0
	for i, f := range append([]*gen.Field{t.ID}, t.Fields...) {
		b.WriteString("    // " + f.FieldComment + "\n")
		tp := fmt.Sprint(f.Type)
		switch tp {
		case "float64":
			tp = "double"
		case "uint8":
			tp = "uint32"
		case "string", "bool":
		case "uint64":
			b.WriteString(`    // @gotags: json:"` + f.Name + `,string,omitempty"` + "\n")
		default:
			tp = "string"
		}
		b.WriteString(fmt.Sprintf("    %s    %s = %d", tp, f.Name, i+1) + ";\n")
		index++
	}
	for x, e := range t.Edges {
		b.WriteString("    // " + e.EdgeComment + "\n")
		b.WriteString("    ")
		if !e.Unique {
			b.WriteString("repeated ")
		}
		b.WriteString(fmt.Sprintf("%s    %s = %d", e.Type.Name, e.Name, index+x+1) + ";\n")
	}
	b.WriteString("}\n\n")
	io.WriteString(p, b.String())
	p.array(t)
}

// array returns grpc message of a type array. The format of the message is:
//
//	Type:
//			<Fields Table>
//
//			<Edges Table>
//
func (p Config) array(t *gen.Type) {
	var b strings.Builder
	b.WriteString("//" + t.Comment() + "查询返回集合\n")
	b.WriteString("message " + t.Name + "Array{\n")
	b.WriteString("    // 便宜量\n")
	b.WriteString("    int32 offset = 1;\n")
	b.WriteString("    // 最大数量\n")
	b.WriteString("    int32 limit = 2;\n")
	b.WriteString("    // 总数\n")
	b.WriteString("    int32 total = 3;\n")
	b.WriteString("    // 结果\n")
	b.WriteString("    repeated " + t.Name + " result = 4;\n")
	b.WriteString("}\n\n")
	io.WriteString(p, b.String())
}
