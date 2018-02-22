package main

import (
  "fmt"
  "os"
  "encoding/gob"
  "strings"
  "unicode"
  "errors"
  "flag"

  "github.com/reiver/go-porterstemmer"
  "github.com/c-bata/go-prompt"
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

  // Defining that a concept is exactly the same (maybe adifferent spelling, etc) as another concept
  "IDENTICAL": RelationKind("IDENTICAL"),

  // Defining that a concept is similar to another concept
  "SYNONYM": RelationKind("SYNONYM"),

  // Defining that a concept is the opposite of another concept
  "ANTONYM": RelationKind("ANTONYM"),

  // Defining that a concept is a more general form of another concept (time is a more general form of an epoch)
  "SUPERSET": RelationKind("SUPERSET"),
  "SUBSET": RelationKind("SUBSET"),
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


func SelectConcept(selector string) *Concept {
  for _, concept := range concepts {
    if concept.Name == porterstemmer.StemString(selector) {
      return concept
    }
  }

  // By default, just look by id.
  for _, concept := range concepts {
    if fmt.Sprintf("%d", concept.Id) == selector {
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
var COMMANDS []Command

func init() {
  COMMANDS = []Command{
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

    Command{Name: "clear", Callback: func(argv []string) error {
      fmt.Print("\033[H\033[2J")
      return nil
    }},
    Command{Name: "exit", Callback: func(argv []string) error {
      os.Exit(0)
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
        if typ, ok = CONCEPT_TYPES[strings.ToUpper(argv[1])]; !ok {
          return errors.New(fmt.Sprintf("Type %s does not exist.", typ))
        }
      }

      conceptId += 1
      concept := &Concept{
        Id: conceptId,
        Name: porterstemmer.StemString(name),
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
          fmt.Printf(" (id=%d)\n", relation.Id)
        }
        if len(concept.Relations) > 0 {
          fmt.Println()
        }
      }
      return nil
    }},
    Command{Name: "rmc", Callback: func(argv []string) error {
      if len(argv) == 0 || len(argv) >= 2 {
        return errors.New("rmc <concept id>")
      }

      concept := SelectConcept(argv[0])
      if concept == nil {
        return errors.New("No such concept found.")
      }
      index := -1
      fmt.Println(concept)

      for ind, c := range concepts {
        if c.Id == concept.Id {
          index = ind
          break
        }
      }

      if index >= 0 {
        fmt.Println(concepts[index:], concepts[:index+1])
        // concepts = append(concepts[index:], concepts[:index+1]...)
        concepts = append(concepts[index:], concepts[:index+1]...)
        fmt.Printf("%d) %s[%s]\n", concept.Id, concept.Name, concept.Type)
        return nil
      } else {
        return errors.New("No such concept found.")
      }
    }},

    // Define a relationship on a concept that defines how it relates to another concept
    Command{Name: "relate", Callback: func(argv []string) error {
      if len(argv) < 2 {
        return errors.New("relate <concept id> <relation type> <concept1> [concept2] ... [conceptn]")
      }
      
      // Find the concept that the relationship is being defined upon
      concept := SelectConcept(argv[0])
      if concept == nil {
        return errors.New(fmt.Sprintf("No such concept found: %d", argv[0]))
      }

      // Find all the concepts that are used as arguments to the relationship
      var relationConcepts []int
      for _, conceptSelector := range argv[2:] {
        concept := SelectConcept(conceptSelector)
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
      concept := SelectConcept(argv[0])
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


    Command{Name: "desc", Callback: func(argv []string) error {
      phrase := strings.Join(argv, " ")
      resultConcepts, err := Describe(phrase)
      if err != nil {
        return err
      }

      for _, concept := range resultConcepts {
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
    Command{Name: "trainc", Callback: func(argv []string) error {
      phrase := strings.Join(argv, " ")
      err := TrainConcept(phrase, 5)
      if err != nil {
        return err
      }
      return nil
    }},
  }
}

func RunCommand(command string, argv []string) error {
  for _, c := range COMMANDS {
    if c.Name == command {
      return c.Callback(argv)
    }
  }
  return errors.New(fmt.Sprintf("No command %s found", command))
}

func completer(d prompt.Document) []prompt.Suggest {
  NO_SUGGESTIONS := []prompt.Suggest{}
  // Ensure that a first word has been typed
  if len(strings.Fields(d.Text)) <= 1 {
    return NO_SUGGESTIONS
  }

  fields := strings.Fields(d.Text)
  command := strings.ToLower(fields[0])

  // Calculate all suggestions for 
  conceptSuggestions := []prompt.Suggest{
  }
  for _, concept := range concepts {
    conceptSuggestions = append(conceptSuggestions, prompt.Suggest{
      Text: concept.Name,
      Description: string(concept.Type),
    })
  }
  
  switch command {
  case "relate":
    if len(fields) == 2 || len(fields) >= 4 {
      return prompt.FilterHasPrefix(conceptSuggestions, d.GetWordBeforeCursor(), true)
    } else {
      return NO_SUGGESTIONS
    }
  case "unrelate":
    return prompt.FilterHasPrefix(conceptSuggestions, d.GetWordBeforeCursor(), true)
  }

	return NO_SUGGESTIONS
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

  for {
    text := prompt.Input("> ", completer)

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


/*
Process:
1. Split the concept up into subconcepts
  - Does the concept exist on its own? If so, then the result from this step is `[concept]`
  - If not, start taking words off of the front to try to form a concept.
  - After each word is added, try to search for a matching concept.
  - If all words are added and no concept is found, then throw an error.
  - Repeat the above process to split the phrase into concepts
2. For each concept:
  - Try to describe it in the terms of other concepts
  - For each relationship within the concept, allow each concept in order to modify an array:

UNION (like synonym, but similar to n concepts combined):
1. Take all concepts passed as args, run each through this process.
2. Take union of all items
3. That becomes the result

EXAMPLE:
1. Take all concepts passed as args, run each through this process.
2. Take the intersection of all items
3. That becomes the result


*/
