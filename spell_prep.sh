#!/bin/bash

# Set the engineering variable
engineering=5

# Declare an array of spells and their respective levels
declare -a spells=(
  "Enlarge Person - Reach:2"
  "Mage Armor - Duration:2"
  "Shocking Grasp - Reach and Intensified:2"
)

# Loop through each spell and call the sg command
for spell in "${spells[@]}"; do
  # Split the spell name and level
  name=$(echo "$spell" | cut -d':' -f1)
  level=$(echo "$spell" | cut -d':' -f2)
  
  # Call the sg command with the name and level
  echo "Calling sg for '$name' with spell_level=$level and engineering=$engineering"
  ./sg "$level" "$engineering"
done
