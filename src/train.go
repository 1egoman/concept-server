package main

import (
  "fmt"
  "net/http"
  "encoding/json"
  "time"
)

type WordnickRelationship struct {
  RelationshipType string `json:"relationshipType"`
  Words []string `json:"words"`
}

var WORDNICK_RELATIONHIP_CONVERSION = map[string]RelationKind{
  "synonym": RELATIONS["SYNONYM"],
  "antonym": RELATIONS["ANTONYM"],
  "variant": RELATIONS["IDENTICAL"],
  // "equivalent"
  // "related-word"
  // "form"
  "hypernym": RELATIONS["SUBSET"],
  "hyponym": RELATIONS["SUPERSET"],
  // "inflected-form"
  // "primary"
  // "same-context"
  // "verb-form"
  // "verb-stem"
}

const WORDNICK_RELATIONSHIPS = "http://api.wordnik.com/v4/word.json/%s/relatedWords?useCanonical=true&limitPerRelationshipType=10&api_key=a2a73e7b926c924fad7001ca3111acd55af2ffabf50eb4ae5"

func GetRelatedConcepts(concept string) ([]WordnickRelationship, error) {
  url := fmt.Sprintf(WORDNICK_RELATIONSHIPS, concept)

  resp, err := http.Get(url)
  if err != nil {
    time.Sleep(1 * time.Second)

    // Make the http request again.
    resp, err = http.Get(url)
    if err != nil {
      time.Sleep(5 * time.Second)

      // Try it one last time
      resp, err = http.Get(url)
      if err != nil {
        return nil, err
      }
    }
  }

  var data []WordnickRelationship

  decoder := json.NewDecoder(resp.Body)
  err = decoder.Decode(&data)
  if err != nil {
    return nil, err
  }

  return data, nil
}

func TrainConcept(concept string, maxdepth int) error {
  if maxdepth == 0 {
    fmt.Println("Max depth reached.")
    return nil
  }

  fmt.Printf("* Training on %s\n", concept)

  // If the concept `concept` doesn't exist, create it.
  baseConcept := SelectConcept(concept)
  if baseConcept == nil {
    err := RunCommand("newc", []string{concept})
    if err != nil {
      return err
    }
    baseConcept = SelectConcept(concept)
  }

  time.Sleep(750 * time.Millisecond)
  data, err := GetRelatedConcepts(concept)
  if err != nil {
    return err
  }

  // Convert from a wordnick concept into a concept that can be used by the system
  for _, wordnickRelationship := range data {
    if relationType, ok := WORDNICK_RELATIONHIP_CONVERSION[wordnickRelationship.RelationshipType]; ok {
      words := wordnickRelationship.Words

      // We now have a relationship of type `relationType` that contains `words`. Next, look through
      // each word and create a concept if one doesn't already exist.
      for _, word := range words {
        fmt.Println("WORD:", word)
        concept := SelectConcept(word)

        // If the word doesn't exist, train on it first.
        if concept == nil {
          err := TrainConcept(word, maxdepth-1)
          if err != nil {
            return err
          }
          concept = SelectConcept(word)
        }

        // Create the relationship on the word.
        fmt.Printf("* Relating %s => %s\n", baseConcept.Name, word)
        RunCommand("relate", []string{
          fmt.Sprintf("%d", baseConcept.Id),
          string(relationType),
          word,
        })

        // Dump data to disk
        fmt.Println("Dumping to disk...")
        RunCommand("dump", []string{"training.db"})
      }
    } else {
      // fmt.Printf("* Skipping wordnick relationship %s\n", wordnickRelationship.RelationshipType);
    }
  }
  return nil
}
