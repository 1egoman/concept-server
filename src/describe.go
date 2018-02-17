package main

import (
  "fmt"
  "strings"
  "errors"
  "github.com/reiver/go-porterstemmer"
)

func resolveConcepts(phraseConcepts []*Concept) []*Concept {
  var returnedRelations []*Concept

  for _, concept := range phraseConcepts {
    for _, relation := range concept.Relations {
      switch relation.Kind {
      case RELATIONS["UNION"]:
        fmt.Println("FOUND RELATION UNION")
        var all []*Concept
        for _, cId := range relation.Concepts {
          for _, c := range concepts {
            if c.Id == cId {
              all = append(all, c)
              break
            }
          }
        }
        returnedRelations = append(returnedRelations, resolveConcepts(all)...)
      case RELATIONS["EXAMPLE"]:
        var all []*Concept
        for _, cId := range relation.Concepts {
          fmt.Println("FOUND RELATION EXAMPLE")
          // Get the concept object
          var concept *Concept
          for _, c := range concepts {
            if c.Id == cId {
              concept = c
              break
            }
          }
          if concept == nil {
            // ERROR: what to do here?
            returnedRelations = append(returnedRelations, all...)
          }
          fmt.Printf("CONCEPT IS %+v\n", concept)

          resolved := resolveConcepts([]*Concept{concept})
          fmt.Printf("RESOLVED %+v\n", resolved)
          all = resolved

          // Do the intersection: remove any items `all` that aren't in `resolved`
          // for index := 0; index < len(all); index++ {
          //   value := all[index]
          //
          //   valueIsInResolved := false
          //   for _, rvalue := range resolved {
          //     if value == rvalue {
          //       valueIsInResolved = true
          //       break
          //     }
          //   }
          //
          //   if !valueIsInResolved {
          //     // Remove the item at `index`
          //     all = append(all[:index], all[index+1:]...)
          //   }
          // }

        }

        returnedRelations = append(returnedRelations, all...)
      }
    }
  }
  return nil
}

func Describe(phrase string) ([]*Concept, error) {
  // Find concepts from phrases
  regularWords := strings.Split(phrase, " ")

  var words []string
  for _, word := range regularWords {
    words = append(words, porterstemmer.StemString(word))
  }

  var phraseConcepts []*Concept
  Outer:
  for len(words) > 0 {
    // Start by trying to match the whole phrase to one concept, and loop through the phrase slowly
    // removing a word from the right side until the phrase matches a known concept.
    for i := len(words); i > 0; i-- {
      potentialPhrase := strings.Join(words[:i], " ")
      concept := SelectConcept(potentialPhrase)

      // Was a concept found?
      if concept != nil {
        // fmt.Printf("* Found concept %+v\n", concept)
        phraseConcepts = append(phraseConcepts, concept)

        // Remove the words that were part of the concept.
        words = words[i:]
        continue Outer
      }
    }

    // No concept found for word
    return nil, errors.New(fmt.Sprintf(
      "No concept could be discerned for the phrase '%s'",
      strings.Join(words, ""),
    ))
  }

  return resolveConcepts(phraseConcepts), nil
}
