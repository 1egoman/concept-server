package main

import (
  "fmt"
  "bufio"
  "os"
  "encoding/gob"
  "strings"
  "unicode"
  "errors"
  "flag"
)

type ConceptType string
var CONCEPT_TYPES map[string]ConceptType = map[string]ConceptType{
  "NOUN": ConceptType("NOUN"),
  "VERB": ConceptType("VERB"),
  "ADJECTIVE": ConceptType("ADJECTIVE"),
  "PRONOUN": ConceptType("PRONOUN"),
}

var conceptId int = 0
type Concept struct {
	Id int
  Name string
  Type ConceptType
  Relations []*Relation
}

type RelationKind string
var RELATIONS map[string]RelationKind = map[string]RelationKind{
  // The base case, a concept that is defined in code
  "KEYWORD": RelationKind("KEYWORD"),

  // Defining that a concept is similar to another concept
  "SYNONYM": RelationKind("SYNONYM"),

  // Defining that a concept is a more general form of another concept (time is a more general form of an epoch)
  "SUPERSET": RelationKind("SUPERSET"),

  "EXAMPLE": RelationKind("EXAMPLE"),

  // Defining that a concept belongs to another concept (a car owns its tires)
  "POSSESIVE": RelationKind("POSSESIVE"),

  // Defining a concept in terms of multipl other concepts
  "UNION": RelationKind("UNION"),
}

var relationId int = 0
type Relation struct {
  Id int
  Kind RelationKind
  Concepts []int
}

type ConceptMap struct {
  Version string
  MaxConceptId int
  MaxRelationId int
  Concepts []*Concept
}
var concepts []*Concept = []*Concept{}


func selectConcept(selector string) *Concept {
  for _, concept := range concepts {
    if concept.Name == selector {
      return concept
    }
  }

  // By default, just look by id.
  for _, concept := range concepts {
    if fmt.Sprintf("%d", concept.Name) == selector {
      return concept
    }
  }

  return nil
}



func hydrateConcepts(filepath string) (*ConceptMap, error) {
  reader, err := os.Open(filepath)
  if err != nil {
    return nil, err
  }
  defer reader.Close()

  unpack := &ConceptMap{}

  dec := gob.NewDecoder(reader)
  err = dec.Decode(unpack)
  if err != nil {
    return nil, err
  }

  return unpack, nil
}

func dumpConcepts(filepath string, conceptmap *ConceptMap) error {
  writer, err := os.Create(filepath)
  if err != nil {
    return err
  }

  enc := gob.NewEncoder(writer)
  err = enc.Encode(conceptmap)
  if err != nil {
    return err
  }

  return nil
}

// Splits a string ito subcomponents by a space, but also takes quotes into account. ie:
// "abc def" => ["abc", "def"]
// "foo 'bar baz'" => ["foo", "bar baz"]
func splitIntoArgv(s string) []string {
	lastQuote := rune(0)
	f := func(c rune) bool {
		switch {
		case c == lastQuote:
			lastQuote = rune(0)
			return false
		case lastQuote != rune(0):
			return false
		case unicode.In(c, unicode.Quotation_Mark):
			lastQuote = c
			return false
		default:
			return unicode.IsSpace(c)
		}
	}

	m := strings.FieldsFunc(s, f)

  // Remove quotes from each section if they exist
  for index, i := range m {
    if len(i) > 0 && unicode.In(rune(i[0]), unicode.Quotation_Mark){
      i = i[1:]
    }
    if len(i) > 0 && unicode.In(rune(i[len(i)-1]), unicode.Quotation_Mark) {
      i = i[:len(i)-1]
    }
    m[index] = i
  }

  return m
}

type Command struct {
  Name string
  Callback func([]string) error
}
var COMMANDS []Command = []Command{
  // Database state-saving commands
  Command{Name: "dump", Callback: func(argv []string) error {
    if len(argv) != 1 {
      return errors.New("Only accepts 1 argument.")
    }
    return dumpConcepts(argv[0], &ConceptMap{
      Version: "v1",
      Concepts: concepts,

      // The current max id that we are at with relations and concepts
      MaxConceptId: conceptId,
      MaxRelationId: relationId,
    })
  }},

  Command{Name: "read", Callback: func(argv []string) error {
    if len(argv) != 1 {
      return errors.New("Only accepts 1 argument.")
    }
    fmt.Printf("Loading %s\n", argv[0])

    conceptmap, err := hydrateConcepts(argv[0])
    if err != nil {
      return err
    }

    fmt.Printf("Version %s\n", conceptmap.Version)
    concepts = conceptmap.Concepts
    conceptId = conceptmap.MaxConceptId
    relationId = conceptmap.MaxRelationId
    return nil
  }},

  // Create new concepts
  Command{Name: "newc", Callback: func(argv []string) error {
    var name string
    var typ ConceptType
    if len(argv) == 0 {
      return errors.New("newc <name> [type]")
    }
    if len(argv) > 2 {
      return errors.New("Too many arguments, expected only 1 or 2.")
    }
    if len(argv) == 1 {
      name = argv[0]
      typ = ""
    }
    if len(argv) == 2 {
      name = argv[0]
      var ok bool
      if typ, ok = CONCEPT_TYPES[argv[1]]; !ok {
        return errors.New(fmt.Sprintf("Type %s does not exist.", typ))
      }
    }

    conceptId += 1
    concept := &Concept{
      Id: conceptId,
      Name: name,
      Type: typ,
    }
    concepts = append(concepts, concept)

    fmt.Printf("%d) %s[%s]\n", concept.Id, concept.Name, concept.Type)
    return nil
  }},
  Command{Name: "lsc", Callback: func(argv []string) error {
    for _, concept := range concepts {
      fmt.Printf("%d) %s[%s]\n", concept.Id, concept.Name, concept.Type)
      for _, relation := range concept.Relations {
        fmt.Printf(" |-> %s of: ", relation.Kind)
        for _, conceptId := range relation.Concepts {
          for _, concept := range concepts {
            if concept.Id == conceptId {
              fmt.Printf("%s ", concept.Name)
            }
          }
        }
        fmt.Printf(" (id=%d)", relation.Id)
      }
      if len(concept.Relations) > 0 {
        fmt.Println()
      }
    }
    return nil
  }},

  // Define a relationship on a concept that defines how it relates to another concept
  Command{Name: "relate", Callback: func(argv []string) error {
    if len(argv) < 3 {
      return errors.New("relate <concept id> <relation type> <concept1> [concept2] ... [conceptn]")
    }
    
    // Find the concept that the relationship is being defined upon
    concept := selectConcept(argv[0])
    if concept == nil {
      return errors.New(fmt.Sprintf("No such concept found: %d", argv[0]))
    }

    // Find all the concepts that are used as arguments to the relationship
    var relationConcepts []int
    for _, conceptSelector := range argv[2:] {
      concept := selectConcept(conceptSelector)
      if concept != nil {
        relationConcepts = append(relationConcepts, concept.Id)
      } else {
        return errors.New(fmt.Sprintf("No such concept found: %d", conceptId))
      }
    }

    relation := strings.ToUpper(argv[1])
    if kind, ok := RELATIONS[relation]; ok {
      relationId += 1
      relation := &Relation{
        Id: relationId,
        Kind: kind,
        Concepts: relationConcepts,
      }

      // Add the relation to the concept.
      concept.Relations = append(concept.Relations, relation)
    } else {
      return errors.New(fmt.Sprintf("No such relation: %s", relation))
    }


    fmt.Printf("%d) %s[%s]\n", concept.Id, concept.Name, concept.Type)
    return nil
  }},
  Command{Name: "unrelate", Callback: func(argv []string) error {
    if len(argv) != 2 {
      return errors.New("unrelate <concept id> <relation id>")
    }
    
    // Find the concept that the relationship is being defined upon
    concept := selectConcept(argv[0])
    if concept == nil {
      return errors.New(fmt.Sprintf("No such concept found: %d", argv[0]))
    }

    // Find the relationship to modify on a concept
    var relationIndex int = -1
    for i, r := range concept.Relations {
      if fmt.Sprintf("%d", r.Id) == argv[1] {
        relationIndex = i
        break
      }
    }
    if relationIndex == -1 {
      return errors.New(fmt.Sprintf("No such relation found in concept %d: %d", argv[0], argv[1]))
    }

    concept.Relations = append(
      concept.Relations[:relationIndex],
      concept.Relations[relationIndex+1:]...
    )

    fmt.Printf("%d) %s[%s]\n", concept.Id, concept.Name, concept.Type)
    return nil
  }},
}

func RunCommand(command string, argv []string) error {
  for _, c := range COMMANDS {
    if c.Name == command {
      return c.Callback(argv)
    }
  }
  return errors.New(fmt.Sprintf("No command %s found", command))
}

func main() {
  // On start, parse some flags to default actions
  var database = flag.String("db", "", "A path to a database file to READ on start.")
  flag.Parse()

  // If a database file was passed, load it on start.
  if *database != "" {
    err := RunCommand("read", []string{*database})
    if err != nil {
      fmt.Printf("err: %s\n", err)
    }
  }

  reader := bufio.NewReader(os.Stdin)
  for {
    fmt.Printf("> ");
    text, _ := reader.ReadString('\n')

    // convert CRLF to LF
    text = strings.Replace(text, "\n", "", -1)

    // convert into commands
    commands := splitIntoArgv(text)
    if len(commands) == 0 {
      continue
    }

    // Run the command that was specified.
    command := strings.ToLower(commands[0])
    argv := commands[1:]
    err := RunCommand(command, argv)
    if err != nil {
      fmt.Printf("err: %s\n", err)
    }
  }
}
