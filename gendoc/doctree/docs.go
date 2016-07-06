// Doctree, which stands for "documentation tree", creates a tree of nodes
// representing the components of a service defined through Protobuf definition
// files. The tree is composed of nodes fulfilling the `Describable` interface,
// with the root node fulfilling the `Doctree` interface. The `Doctree`
// interface is a superset of the `Describable` interface.
//
// The main entrypoint for the Doctree package is the `New` function, which
// takes a Protobuf `CodeGeneratorRequest` struct and creates a Doctree
// representing all the documentation from the `CodeGeneratorRequest`.
//
// For a larger explanation of how and why Doctree is structured the way it is,
// see the comment for the 'associateComments' function in the source code of
// the 'associate_comments.go' file.
package doctree

import (
	"fmt"
	"strings"
)

// prindent is a utility function for creating a formatted string with a given
// amount of indentation.
func prindent(depth int, format string, args ...interface{}) string {
	s := ""
	for i := 0; i < depth; i++ {
		s += "    "
	}
	return s + fmt.Sprintf(format, args...)
}

// strRepeat takes a string and an int `n` and returns a string representing
// the input repeated `n` times.
func strRepeat(in string, count int) string {
	rv := ""
	for ; count > 0; count-- {
		rv += in
	}
	return rv
}

// Describable offers an interface for traversing a Doctree and finding
// information from the nodes within it.
type Describable interface {
	// The "Name" of this describable
	GetName() string
	SetName(string)
	// GetDescription returns the description of this describable
	GetDescription() string
	SetDescription(string)
	// describe causes a Describable to generate a string representing itself.
	// The integer argument is used as the 'depth' that this Describable sits
	// within a tree of Describable structs, allowing it to print it's
	// information with proper indentation. If called recursively, allows for
	// printing of a structured tree-style view of a tree of Describables.
	describe(int) string
	describeMarkdown(int) string
	// GetByName allows one to query a Describable to see if it has a child
	// Describable in any of it's collections.
	GetByName(string) Describable
}

// Doctree is the root interface for this package, and is chiefly implemented
// by MicroserviceDefinition. See MicroserviceDefinition for further
// documentation on these Methods.
type Doctree interface {
	Describable
	SetComment([]string, string)
	String() string
	Markdown() string
}

// describable is a  concrete implementation of the `Describable` interface, to
// allow for nice convenient inheritance with concrete default methods.
type describable struct {
	Name        string
	Description string
}

func (self *describable) GetName() string {
	return self.Name
}

func (self *describable) SetName(s string) {
	self.Name = s
}

func (self *describable) describe(depth int) string {
	rv := prindent(depth, "Name: %v\n", self.Name)
	rv += prindent(depth, "Desc: %v\n", self.Description)
	return rv
}

func (self *describable) describeMarkdown(depth int) string {
	rv := prindent(0, "%v %v\n\n", strRepeat("#", depth), self.Name)
	if len(self.Description) > 1 {
		rv += prindent(0, "%v\n\n", self.Description)
	}
	return rv
}

func (self *describable) GetDescription() string {
	return self.Description
}

func (self *describable) SetDescription(d string) {
	// When setting a description, clean it up
	self.Description = scrubComments(d)
}

func (self *describable) GetByName(s string) Describable {
	return nil
}

// MicroserviceDefinition is the root node for any particular `Doctree`
type MicroserviceDefinition struct {
	describable
	Files []*ProtoFile
}

func (self *MicroserviceDefinition) describe(depth int) string {
	rv := self.describable.describe(depth)
	for idx, file := range self.Files {
		rv += prindent(depth, "File %v:\n", idx)
		rv += file.describe(depth + 1)
	}
	return rv
}

func (self *MicroserviceDefinition) describeMarkdown(depth int) string {
	rv := doc_css
	rv += self.describable.describeMarkdown(depth)
	//rv += prindent(0, "%v %v\n\n", strRepeat("#", depth), "Files")
	for _, file := range self.Files {
		rv += file.describeMarkdown(depth + 1)
	}
	return rv
}

// GetByName returns any ProtoFile structs it my have with a matching `Name`.
func (self *MicroserviceDefinition) GetByName(name string) Describable {
	for _, file := range self.Files {
		if file.Name == name {
			return file
		}
	}
	return nil
}

// SetComment changes the node at the given 'name-path' to have a description
// of `comment_body`. `name-path` is a series of names of describable objects
// each found within the previous, accessed by recursively calling `GetByName`
// on the result of the last call, beginning with this MicroserviceDefinition.
// Once the final Describable object is found, the `description` field of that
// struct is set to `comment_body`.
func (self *MicroserviceDefinition) SetComment(namepath []string, comment_body string) {
	var cur_node Describable
	cur_node = self
	for _, name := range namepath {
		new_node := cur_node.GetByName(name)
		if new_node == nil {
			panic(fmt.Sprintf("New node is nil, namepath: '%v' cur_node: '%v'\n", namepath, cur_node))
		}
		cur_node = new_node
	}
	cur_node.SetDescription(comment_body)
}

// String kicks off the recursive call to `describe` within the tree of
// Describables, returning a string showing the structured view of the tree.
func (self *MicroserviceDefinition) String() string {
	return self.describe(0)
}

func (self *MicroserviceDefinition) Markdown() string {
	return self.describeMarkdown(1)
}

type ProtoFile struct {
	describable
	Messages []*ProtoMessage
	Enums    []*ProtoEnum
	Services []*ProtoService
}

func (self *ProtoFile) describe(depth int) string {
	rv := self.describable.describe(depth)
	for idx, svc := range self.Services {
		rv += prindent(depth, "Service %v:\n", idx)
		rv += svc.describe(depth + 1)
	}
	for idx, msg := range self.Messages {
		rv += prindent(depth, "Message %v:\n", idx)
		rv += msg.describe(depth + 1)
	}
	for idx, enum := range self.Enums {
		rv += prindent(depth, "Enum %v:\n", idx)
		rv += enum.describe(depth + 1)
	}
	return rv
}

func (self *ProtoFile) describeMarkdown(depth int) string {
	rv := self.describable.describeMarkdown(depth)

	if len(self.Messages) > 0 {
		rv += prindent(0, "%v %v\n\n", strRepeat("#", depth+1), "Messages")
		for _, msg := range self.Messages {
			rv += msg.describeMarkdown(depth + 2)
		}
	}

	if len(self.Enums) > 0 {
		rv += prindent(0, "%v %v\n\n", strRepeat("#", depth+1), "Enums")
		for _, enum := range self.Enums {
			rv += enum.describeMarkdown(depth + 2)
		}
	}

	if len(self.Services) > 0 {
		rv += prindent(0, "%v %v\n\n", strRepeat("#", depth+1), "Services")
		for _, svc := range self.Services {
			rv += svc.describeMarkdown(depth + 2)
		}
	}
	return rv
}

func (self *ProtoFile) GetByName(name string) Describable {
	for _, msg := range self.Messages {
		if msg.GetName() == name {
			return msg
		}
	}
	for _, enum := range self.Enums {
		if enum.GetName() == name {
			return enum
		}
	}
	for _, svc := range self.Services {
		if svc.GetName() == name {
			return svc
		}
	}
	return nil
}

type ProtoMessage struct {
	describable
	Fields []*MessageField
}

func (self *ProtoMessage) describe(depth int) string {
	rv := self.describable.describe(depth)
	for idx, field := range self.Fields {
		rv += prindent(depth, "Field %v:\n", idx)
		rv += field.describe(depth + 1)
	}
	return rv
}

func (self *ProtoMessage) describeMarkdown(depth int) string {
	// Embed an anchor above this title, to allow for things to link to it. The
	// 'name' of this anchor link is just the name of this ProtoMessage. This
	// may not reliably create unique 'name's in all cases, but I've not
	// encountered any problems with this aproach thus far so I'm keeping it.
	rv := `<a name="` + self.Name + `"></a>` + "\n\n"
	rv += prindent(0, "%v %v\n\n", strRepeat("#", depth), self.Name)
	if len(self.Description) > 1 {
		rv += prindent(0, "%v\n\n", self.Description)
	}

	// If there's no fields, avoid printing an empty table by short-circuiting
	if len(self.Fields) < 1 {
		rv += "\n"
		return rv
	}

	rv += "| Name | Type | Field Number | Description|\n"
	rv += "| ---- | ---- | ------------ | -----------|\n"
	for _, f := range self.Fields {
		safe_desc := f.GetDescription()
		safe_desc = strings.Replace(safe_desc, "\n", "", -1)
		rv += fmt.Sprintf("| %v | %v | %v | %v |\n", f.GetName(), f.Type.Name, f.Number, safe_desc)
	}
	rv += "\n"
	return rv

}

func (self *ProtoMessage) GetByName(name string) Describable {
	for _, field := range self.Fields {
		if field.GetName() == name {
			return field
		}
	}
	return nil
}

type MessageField struct {
	describable
	Type   FieldType
	Number int
}

func (self *MessageField) describe(depth int) string {
	rv := self.describable.describe(depth)
	rv += prindent(depth, "Number: %v\n", self.Number)
	rv += prindent(depth, "Type:\n")
	rv += self.Type.describe(depth + 1)
	return rv
}

type ProtoEnum struct {
	describable
	Values []*EnumValue
}

func (self *ProtoEnum) describe(depth int) string {
	rv := self.describable.describe(depth)
	for idx, val := range self.Values {
		rv += prindent(depth, "Value %v:\n", idx)
		rv += val.describe(depth + 1)
	}
	return rv
}

func (self *ProtoEnum) describeMarkdown(depth int) string {
	rv := self.describable.describeMarkdown(depth)
	rv += "| Number | Name |\n"
	rv += "| ------ | ---- |\n"
	for _, val := range self.Values {
		rv += prindent(0, "| %v | %v |\n", val.Number, val.Name)
	}
	rv += "\n\n"
	return rv
}

type EnumValue struct {
	describable
	Number int
}

func (self *EnumValue) describe(depth int) string {
	rv := self.describable.describe(depth)
	rv += prindent(depth, "Number: %v\n", self.Number)
	return rv
}

type FieldType struct {
	describable
	Enum *ProtoEnum
}

type ProtoService struct {
	describable
	Methods []*ServiceMethod
}

func (self *ProtoService) describe(depth int) string {
	rv := self.describable.describe(depth)
	for idx, meth := range self.Methods {
		rv += prindent(depth, "Method %v:\n", idx)
		rv += meth.describe(depth + 1)
	}
	return rv
}

func (self *ProtoService) describeMarkdown(depth int) string {
	rv := self.describable.describeMarkdown(depth)

	rv += "| Name | Request Type | Response Type | Description|\n"
	rv += "| ---- | ---- | ------------ | -----------|\n"
	for _, meth := range self.Methods {
		//rv += meth.describeMarkdown(depth + 2)
		rv += prindent(0, "| %v | %v | %v | %v |\n", meth.GetName(), meth.RequestType.GetName(), meth.ResponseType.GetName(), meth.GetDescription())
	}
	//rv += prindent(0, "%v %v\n\n", strRepeat("#", depth+1), "Methods")
	return rv
}

func (self *ProtoService) GetByName(name string) Describable {
	for _, meth := range self.Methods {
		if meth.GetName() == name {
			return meth
		}
	}
	return nil
}

type ServiceMethod struct {
	describable
	RequestType  *ProtoMessage
	ResponseType *ProtoMessage
	HttpBindings []*ServiceHttpBinding
}

func (self *ServiceMethod) describe(depth int) string {
	rv := self.describable.describe(depth)
	rv += prindent(depth, "RequestType: %v\n", self.RequestType.GetName())
	rv += prindent(depth, "ResponseType: %v\n", self.ResponseType.GetName())
	rv += prindent(depth, "ServiceHttpBinding:\n")

	for _, bind := range self.HttpBindings {
		rv += bind.describe(depth + 1)
	}
	return rv
}

func (self *ServiceMethod) describeMarkdown(depth int) string {
	rv := self.describable.describeMarkdown(depth)

	rv += prindent(0, "[RequestType: %v](#%v)\n\n", self.RequestType.GetName(), self.RequestType.GetName())
	rv += prindent(0, "[ResponseType: %v](#%v)\n\n", self.ResponseType.GetName(), self.ResponseType.GetName())

	for _, bind := range self.HttpBindings {
		rv += bind.describeMarkdown(depth + 1)
	}

	return rv
}

func (self *ServiceMethod) GetByName(name string) Describable {
	if name == self.RequestType.GetName() {
		return self.RequestType
	}
	if name == self.ResponseType.GetName() {
		return self.ResponseType
	}
	return nil
}

type ServiceHttpBinding struct {
	describable
	Fields []*BindingField
}

func (self *ServiceHttpBinding) describe(depth int) string {
	rv := self.describable.describe(depth)
	for _, field := range self.Fields {
		rv += field.describe(depth + 1)
	}
	return rv
}

func (self *ServiceHttpBinding) describeMarkdown(depth int) string {
	rv := ""

	for _, field := range self.Fields {
		rv += prindent(0, "- '%v' : '%v'\n", field.Kind, field.Value)
		if field.GetDescription() != "" {
			rv += prindent(1, "- %v\n", field.GetDescription())
		}
	}
	rv += "\n"

	return rv
}

type BindingField struct {
	describable
	Kind  string
	Value string
}

func (self *BindingField) describe(depth int) string {
	rv := self.describable.describe(depth)
	rv += prindent(depth, "Kind: %v\n", self.Kind)
	rv += prindent(depth, "Value: %v\n", self.Value)
	return rv
}
