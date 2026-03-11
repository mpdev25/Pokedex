package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/mpdev25/pokedexcli/internal/pokecache"
)

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	cfg := &config{
		Cache: pokecache.NewCache(5 * time.Minute),
	}
	for {
		fmt.Print("Pokedex > ")
		if !scanner.Scan() {
			break
		}
		line := scanner.Text()
		words := cleanInput(line)

		if len(words) == 0 {
			continue
		}
		commandName := words[0]
		if cmd, exists := commands[commandName]; exists {
			err := cmd.callback(cfg, words[1:])
			if err != nil {
				fmt.Printf("Error executing command %s: %v\n", commandName, err)
			}
		} else {
			fmt.Println("Unknown command")
		}
	}
	fmt.Print("Pokedex > ")
}

type cliCommand struct {
	name        string
	description string
	callback    func(*config, []string) error
}

type config struct {
	Next          string `json:"next"`
	Previous      string `json:"previous"`
	Cache         *pokecache.Cache
	caughtPokemon map[string]Pokemon
}

type LocationAreaResponse struct {
	Count    int     `json:"count"`
	Next     *string `json:"next"`
	Previous *string `json:"previous"`
	Results  []struct {
		Name     string  `json:"name"`
		URL      string  `json:"url"`
		Previous *string `json:"previous"`
	} `json:"results"`
}
type LocationAreaExplored struct {
	Name              string `json:"name"`
	PokemonEncounters []struct {
		Pokemon struct {
			Name string `json:"name"`
		} `json:"pokemon"`
	} `json:"pokemon_encounters"`
}

type Pokemon struct {
	Name           string `json:"name"`
	BaseExperience int    `json:"base_experience"`
	Height         int    `json:"height"`
	Weight         int    `json:"weight"`
	Stats          []struct {
		BaseStat int `json:"base_stat"`
		Stat     struct {
			Name string `json:"name"`
		} `json:"stat"`
	} `json:"stats"`
	Types []struct {
		Type struct {
			Name string `json:"name"`
		} `json:"type"`
	} `json:"types"`
}

var commands map[string]cliCommand

func commandExit(cfg *config, args []string) error {
	print("Closing the Pokedex... Goodbye!")
	os.Exit(0)
	return nil
}
func commandHelp(cfg *config, args []string) error {
	fmt.Println("Welcome to the Pokedex!")
	fmt.Print("Usage: \n\n")
	for _, cmd := range commands {
		fmt.Printf("%s: %s\n", cmd.name, cmd.description)

	}
	return nil
}

func fetchLocationData(url string, cfg *config) ([]byte, error) {
	if data, found := cfg.Cache.Get(url); found {
		fmt.Println(url)
		return data, nil
	}
	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode > 299 {
		return nil, fmt.Errorf("Response failed with status code %d", res.StatusCode)
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	cfg.Cache.Add(url, body)
	return body, nil
}

func fetchPokemonData(url string, cfg *config) ([]byte, error) {

	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode > 299 {
		return nil, fmt.Errorf("Response failed with status code %d", res.StatusCode)
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func commandMap(cfg *config, args []string) error {
	url := "https://pokeapi.co/api/v2/location-area/"
	if cfg.Next != "" {
		url = cfg.Next
	}
	res, err := fetchLocationData(url, cfg)
	if err != nil {
		return err
	}

	var locations LocationAreaResponse
	err = json.Unmarshal(res, &locations)
	if err != nil {
		return err
	}
	if locations.Next != nil {
		cfg.Next = *locations.Next
	} else {
		cfg.Next = ""
	}
	if locations.Previous != nil {
		cfg.Previous = *locations.Previous
	} else {
		cfg.Previous = ""
	}
	for _, loc := range locations.Results {
		fmt.Println(loc.Name)
	}
	return nil
}

func commandMapb(cfg *config, args []string) error {
	if cfg.Previous == "" {
		fmt.Println("You are on the first page")
		return nil
	}
	url := "https://pokeapi.co/api/v2/location-area/"
	if cfg.Previous != "" {
		url = cfg.Previous
	}
	res, err := fetchLocationData(url, cfg)
	if err != nil {
		return err
	}

	var locations LocationAreaResponse
	err = json.Unmarshal(res, &locations)
	if err != nil {
		return err
	}
	if locations.Previous != nil {
		cfg.Previous = *locations.Previous
	} else {
		cfg.Previous = ""
	}
	if locations.Next != nil {
		cfg.Next = *locations.Next
	} else {
		cfg.Next = ""
	}
	for _, loc := range locations.Results {
		fmt.Println(loc.Name)
	}
	return nil
}
func commandExplore(cfg *config, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("explore requires an area name")
	}
	areaName := args[0]
	fmt.Printf("Exploring %s...\n", areaName)
	url := fmt.Sprintf("https://pokeapi.co/api/v2/location-area/%s", areaName)
	res, err := fetchLocationData(url, cfg)
	if err != nil {
		return err
	}
	var areaData LocationAreaExplored
	err = json.Unmarshal(res, &areaData)
	if err != nil {
		return err
	}
	fmt.Println("Found Pokemon:")
	for _, encounter := range areaData.PokemonEncounters {
		fmt.Printf(" - %s\n", encounter.Pokemon.Name)
	}
	return nil
}

func commandCatch(cfg *config, args []string) error {

	if len(args) == 0 {
		return fmt.Errorf("catch requires a Pokemon name")
	}
	pokemonName := args[0]
	fmt.Printf("Throwing a Pokeball at %s...\n", pokemonName)
	url := fmt.Sprintf("https://pokeapi.co/api/v2/pokemon/%s", pokemonName)
	res, err := fetchPokemonData(url, cfg)
	if err != nil {
		return err
	}
	var pokemonData Pokemon
	err = json.Unmarshal(res, &pokemonData)
	if err != nil {
		return fmt.Errorf("failed to unmarshal pokemon data: %w", err)
	}

	catchRate := 100 - pokemonData.BaseExperience/5
	rand.Seed(time.Now().UnixNano())
	roll := rand.Intn(100)
	if roll < catchRate {
		fmt.Printf("You caught %s!\n", pokemonName)
		if cfg.caughtPokemon == nil {
			cfg.caughtPokemon = make(map[string]Pokemon)
		}
		cfg.caughtPokemon[pokemonName] = pokemonData
		fmt.Printf("%s has been added to your Pokedex.\n", pokemonName)
		fmt.Println("You can now inspect this Pokemon.")
	} else {
		fmt.Printf("%s broke free! Try again.\n", pokemonName)
	}
	return nil
}

func commandInspect(cfg *config, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("inspect requires a PokemNaon name")
	}
	pokemonName := args[0]

	pokemon, exists := cfg.caughtPokemon[pokemonName]
	if !exists {
		return fmt.Errorf("you have not caught this Pokemon")
	}
	fmt.Printf("Name: %s\n", pokemon.Name)
	fmt.Printf("Height: %d\n", pokemon.Height)
	fmt.Printf("Weight: %d\n", pokemon.Weight)
	fmt.Println("Stats:")
	for _, s := range pokemon.Stats {
		fmt.Printf(" - %s: %d\n", s.Stat.Name, s.BaseStat)
	}
	fmt.Println("Types:")
	for _, t := range pokemon.Types {
		fmt.Printf(" %s\n", t.Type.Name)
	}
	return nil
}

func commandPokedex(cfg *config, args []string) error {
	fmt.Println("Your Pokedex:")
	for _, pokemon := range cfg.caughtPokemon {
		fmt.Printf("- %s\n", pokemon.Name)
	}
	return nil
}

func init() {
	commands = map[string]cliCommand{
		"exit": {
			name:        "exit",
			description: "Exit the Pokedex",
			callback:    commandExit,
		},
		"help": {
			name:        "help",
			description: "Displays a help message",
			callback:    commandHelp,
		},
		"map": {
			name:        "map",
			description: "Display 20 location areas in the Pokemon world",
			callback:    commandMap,
		},
		"mapb": {
			name:        "mapb",
			description: "Display previous 20 location areas in the Pokemon world",
			callback:    commandMapb,
		},
		"explore": {
			name:        "explore",
			description: "Display all of the Pokemon at the current location",
			callback:    commandExplore,
		},
		"catch": {
			name:        "catch",
			description: "Attempt to catch a Pokemon",
			callback:    commandCatch,
		},
		"inspect": {
			name:        "inspect",
			description: "Inspect a Pokemon",
			callback:    commandInspect,
		},
		"pokedex": {
			name:        "pokedex",
			description: "View your Pokemon collection",
			callback:    commandPokedex,
		},
	}
}
