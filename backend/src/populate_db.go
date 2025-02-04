package main

import (
	"database/sql"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
)

var tables = []FiveETable{
	{
		"ability_scores",
		[]string{
			"index",
			"name",
			"full_name",
			"desc",
			"skills",
			"url",
		},
		"5e-SRD-Ability-Scores.json",
	},
	{
		"alignments",
		[]string{
			"index",
			"name",
			"abbreviation",
			"desc",
			"url",
		},
		"5e-SRD-Alignments.json",
	},
	{
		"backgrounds",
		[]string{
			"index",
			"name",
			"starting_proficiencies",
			"language_options",
			"starting_equipment",
			"starting_equipment_options",
			"feature",
			"personality_traits",
			"ideals",
			"bonds",
			"flaws",
			"url",
		},
		"5e-SRD-Backgrounds.json",
	},
	{
		"classes",
		[]string{
			"index",
			"name",
			"hit_die",
			"proficiency_choices",
			"proficiencies",
			"saving_throws",
			"starting_equipment",
			"starting_equipment_options",
			"class_levels",
			"multi_classing",
			"subclasses",
			"spellcasting",
			"spells",
			"url",
		},
		"5e-SRD-Classes.json",
	},
	{
		"conditions",
		[]string{
			"index",
			"name",
			"desc",
			"url",
		},
		"5e-SRD-Conditions.json",
	},
	{
		"damage_types",
		[]string{
			"index",
			"name",
			"desc",
			"url",
		},
		"5e-SRD-Damage-Types.json",
	},
	{
		"equipment_categories",
		[]string{
			"index",
			"name",
			"equipment",
			"url",
		},
		"5e-SRD-Equipment-Categories.json",
	},
	{
		"rpg_gear",
		[]string{
			"index",
			"name",
			"equipment_category",
			"gear_category",
			"weight",
			"cost",
			"desc",
			"quantity",
			"url",
		},
		"5e-SRD-Rpg-Gear.json",
	},
	{
		"gear",
		[]string{
			"index",
			"name",
			"equipment_category",
			"armor_category",
			"armor_class",
			"str_minimum",
			"stealth_disadvantage",
			"weight",
			"cost",
			"url",
		},
		"5e-SRD-Gear.json",
	},
	{
		"tools",
		[]string{
			"index",
			"name",
			"equipment_category",
			"tool_category",
			"weight",
			"cost",
			"desc",
			"url",
		},
		"5e-SRD-Tools.json",
	},
	{
		"mounts",
		[]string{
			"index",
			"name",
			"equipment_category",
			"vehicle_category",
			"cost",
			"weight",
			"desc",
			"speed",
			"capacity",
			"url",
		},
		"5e-SRD-Mounts.json",
	},
	{
		"weapons",
		[]string{
			"two_handed_damage",
			"special",
			"index",
			"throw_range",
			"name",
			"equipment_category",
			"weapon_category",
			"weapon_range",
			"category_range",
			"cost",
			"damage",
			"range",
			"weight",
			"properties",
			"url",
		},
		"5e-SRD-Weapons.json",
	},
	{
		"feats",
		[]string{
			"index",
			"name",
			"prerequisites",
			"desc",
			"url",
		},
		"5e-SRD-Feats.json",
	},
	{
		"features",
		[]string{
			"index",
			"reference",
			"class",
			"subclass",
			"name",
			"level",
			"prerequisites",
			"feature_specific",
			"parent",
			"desc",
			"url",
		},
		"5e-SRD-Features.json",
	},
	{
		"languages",
		[]string{
			"index",
			"name",
			"type",
			"typical_speakers",
			"script",
			"desc",
			"url",
		},
		"5e-SRD-Languages.json",
	},
	{
		"levels",
		[]string{
			"level",
			"ability_score_bonuses",
			"prof_bonus",
			"features",
			"class_specific",
			"index",
			"class",
			"url",
			"spellcasting",
			"subclass",
			"subclass_specific",
		},
		"5e-SRD-Levels.json",
	},
	{
		"magic_items",
		[]string{
			"index",
			"name",
			"equipment_category",
			"rarity",
			"variants",
			"variant",
			"desc",
			"image",
			"url",
		},
		"5e-SRD-Magic-Items.json",
	},
	{
		"magic_schools",
		[]string{
			"index",
			"name",
			"desc",
			"url",
		},
		"5e-SRD-Magic-Schools.json",
	},
	{
		"monsters",
		[]string{
			"index",
			"desc",
			"subtype",
			"reactions",
			"forms",
			"name",
			"size",
			"type",
			"alignment",
			"armor_class",
			"hit_points",
			"hit_dice",
			"hit_points_roll",
			"speed",
			"strength",
			"dexterity",
			"constitution",
			"intelligence",
			"wisdom",
			"charisma",
			"proficiencies",
			"damage_vulnerabilities",
			"damage_resistances",
			"damage_immunities",
			"condition_immunities",
			"senses",
			"languages",
			"challenge_rating",
			"proficiency_bonus",
			"xp",
			"special_abilities",
			"actions",
			"legendary_actions",
			"image",
			"url",
		},
		"5e-SRD-Monsters.json",
	},
	{
		"proficiencies",
		[]string{
			"index",
			"type",
			"name",
			"classes",
			"races",
			"url",
			"reference",
		},
		"5e-SRD-Proficiencies.json",
	},
	{
		"races",
		[]string{
			"language_options",
			"ability_bonus_options",
			"index",
			"name",
			"speed",
			"ability_bonuses",
			"alignment",
			"age",
			"size",
			"size_description",
			"starting_proficiencies",
			"starting_proficiency_options",
			"languages",
			"language_desc",
			"traits",
			"subraces",
			"url",
		},
		"5e-SRD-Races.json",
	},
	{
		"rule_sections",
		[]string{
			"name",
			"index",
			"desc",
			"url",
		},
		"5e-SRD-Rule-Sections.json",
	},
	{
		"rules",
		[]string{
			"name",
			"index",
			"desc",
			"subsections",
			"url",
		},
		"5e-SRD-Rules.json",
	},
	{
		"skills",
		[]string{
			"index",
			"name",
			"desc",
			"ability_score",
			"url",
		},
		"5e-SRD-Skills.json",
	},
	{
		"spells",
		[]string{
			"dc",
			"heal_at_slot_level",
			"area_of_effect",
			"index",
			"name",
			"desc",
			"higher_level",
			"range",
			"components",
			"material",
			"ritual",
			"duration",
			"concentration",
			"casting_time",
			"level",
			"attack_type",
			"damage",
			"school",
			"classes",
			"subclasses",
			"url",
		},
		"5e-SRD-Spells.json",
	},
	{
		"subclasses",
		[]string{
			"spells",
			"index",
			"class",
			"name",
			"subclass_flavor",
			"desc",
			"subclass_levels",
			"url",
		},
		"5e-SRD-Subclasses.json",
	},
	{
		"subraces",
		[]string{
			"language_options",
			"index",
			"langauge_options",
			"name",
			"race",
			"desc",
			"ability_bonuses",
			"starting_proficiencies",
			"languages",
			"racial_traits",
			"url",
		},
		"5e-SRD-Subraces.json",
	},
	{
		"traits",
		[]string{
			"proficiency_choices",
			"trait_specific",
			"parent",
			"language_options",
			"index",
			"races",
			"subraces",
			"name",
			"desc",
			"proficiencies",
			"url",
		},
		"5e-SRD-Traits.json",
	},
	{
		"weapon_properties",
		[]string{
			"index",
			"name",
			"desc",
			"url",
		},
		"5e-SRD-Weapon-Properties.json",
	},
}

type FiveETable struct {
	name    string
	mapping []string
	file    string
}

func convert_key(key string) string {
	switch key {
	case "index":
		return "_index"
	case "desc":
		return "_desc"
	default:
		return key
	}
}

func safeSQLValue(value interface{}) string {
	switch v := value.(type) {
	case string:
		// Escape single quotes
		return "'" + strings.ReplaceAll(v, "'", "''") + "'"
	default:
		// Convert non-string values (e.g., maps, slices) to JSON
		jsonValue, err := json.Marshal(v)
		if err != nil {
			log.Printf("Error converting value to JSON: %v", err)
			return "NULL" // Fail gracefully
		}
		return "'" + strings.ReplaceAll(string(jsonValue), "'", "''") + "'" // Store as JSON string
	}
}

func createTable(table *FiveETable, db *sql.DB) error {
	log := log.WithField("table", table.name)

	query := "CREATE TABLE " + table.name + " ("
	for i, key := range table.mapping {
		key = convert_key(key)
		if i != 0 {
			query += ", "
		}
		query += key + " TEXT"
	}
	query += ");"

	log.WithField("query", query).Debug("Executing query")
	_, err := db.Exec(query)
	if err != nil {
		log.WithError(err).Error("Failed to create table")
		return err
	}

	log.Info("Created table")

	return nil
}

func insert(table *FiveETable, db *sql.DB, data []map[string]interface{}) error {
	var err error

	for _, row := range data {
		log := log.WithFields(log.Fields{
			"table": table,
			"row":   row,
		})

		for key := range row {
			found := false
			for _, k := range table.mapping {
				if key == k {
					found = true
				}
			}
			if !found {
				log.WithFields(logrus.Fields{
					"key": key,
				}).Fatalf("Extra key in row")
			}
		}

		query := "INSERT INTO " + table.name + " ("
		values := "VALUES ("
		for i, key := range table.mapping {
			if i != 0 {
				query += ", "
				values += ", "
			}
			query += convert_key(key)
			if row[key] == nil {
				log.WithFields(logrus.Fields{
					"key":   key,
					"value": row[key],
				}).Debug("NULL value")
				values += "NULL"
			} else {
				values += safeSQLValue(row[key])
			}
		}
		query += ") " + values + ");"

		_, err := db.Exec(query)
		if err != nil {
			log.WithError(err).Error("Failed to insert row")
			return err
		}

		log.Debug("Inserted row")
	}

	return err
}

func populate(db *sql.DB) error {
	dir := "5e_data"

	for _, table := range tables {
		log := log.WithField("table", table.name)
		file := filepath.Join(dir, table.file)

		jsonFile, err := os.Open(file)
		defer jsonFile.Close()

		jsonData, err := io.ReadAll(jsonFile)
		if err != nil {
			log.WithError(err).Error("Failed to read JSON file")
			return err
		}

		var data []map[string]interface{}
		if err := json.Unmarshal(jsonData, &data); err != nil {
			log.WithError(err).Error("Failed to unmarshal JSON data")
			return err
		}

		if err := createTable(&table, db); err != nil {
			log.WithError(err).Error("Failed to create table")
			return err
		}

		if err := insert(&table, db, data); err != nil {
			log.WithError(err).Error("Failed to insert data")
			return err
		}
	}

	return nil
}
