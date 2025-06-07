package schema

import "google.golang.org/genai"

// Schema 是 OpenAPI 3.0 的 Schema 类型，这里直接使用 genai.Schema 类型
type Schema = genai.Schema

// genai.Schema 的定义
// type Schema struct {
//     // Optional. The value should be validated against any (one or more) of the subschemas
//     // in the list.
//     AnyOf []*Schema `json:"anyOf,omitempty"`
//     // Optional. Default value of the data.
//     Default any `json:"default,omitempty"`
//     // Optional. The description of the data.
//     Description string `json:"description,omitempty"`
//     // Optional. Possible values of the element of primitive type with enum format. Examples:
//     // 1. We can define direction as : {type:STRING, format:enum, enum:["EAST", NORTH",
//     // "SOUTH", "WEST"]} 2. We can define apartment number as : {type:INTEGER, format:enum,
//     // enum:["101", "201", "301"]}
//     Enum []string `json:"enum,omitempty"`
//     // Optional. Example of the object. Will only populated when the object is the root.
//     Example any `json:"example,omitempty"`
//     // Optional. The format of the data. Supported formats: for NUMBER type: "float", "double"
//     // for INTEGER type: "int32", "int64" for STRING type: "email", "byte", etc
//     Format string `json:"format,omitempty"`
//     // Optional. SCHEMA FIELDS FOR TYPE ARRAY Schema of the elements of Type.ARRAY.
//     Items *Schema `json:"items,omitempty"`
//     // Optional. Maximum number of the elements for Type.ARRAY.
//     MaxItems *int64 `json:"maxItems,omitempty"`
//     // Optional. Maximum length of the Type.STRING
//     MaxLength *int64 `json:"maxLength,omitempty"`
//     // Optional. Maximum number of the properties for Type.OBJECT.
//     MaxProperties *int64 `json:"maxProperties,omitempty"`
//     // Optional. Maximum value of the Type.INTEGER and Type.NUMBER
//     Maximum *float64 `json:"maximum,omitempty"`
//     // Optional. Minimum number of the elements for Type.ARRAY.
//     MinItems *int64 `json:"minItems,omitempty"`
//     // Optional. SCHEMA FIELDS FOR TYPE STRING Minimum length of the Type.STRING
//     MinLength *int64 `json:"minLength,omitempty"`
//     // Optional. Minimum number of the properties for Type.OBJECT.
//     MinProperties *int64 `json:"minProperties,omitempty"`
//     // Optional. Minimum value of the Type.INTEGER and Type.NUMBER.
//     Minimum *float64 `json:"minimum,omitempty"`
//     // Optional. Indicates if the value may be null.
//     Nullable *bool `json:"nullable,omitempty"`
//     // Optional. Pattern of the Type.STRING to restrict a string to a regular expression.
//     Pattern string `json:"pattern,omitempty"`
//     // Optional. SCHEMA FIELDS FOR TYPE OBJECT Properties of Type.OBJECT.
//     Properties map[string]*Schema `json:"properties,omitempty"`
//     // Optional. The order of the properties. Not a standard field in open API spec. Only
//     // used to support the order of the properties.
//     PropertyOrdering []string `json:"propertyOrdering,omitempty"`
//     // Optional. Required properties of Type.OBJECT.
//     Required []string `json:"required,omitempty"`
//     // Optional. The title of the Schema.
//     Title string `json:"title,omitempty"`
//     // Optional. The type of the data.
//     Type Type `json:"type,omitempty"`
// }
