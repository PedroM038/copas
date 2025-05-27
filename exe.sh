#!/bin/bash
# filepath: /home/pedro-marques/Documentos/Projects/copas/start_all.sh

echo "Iniciando todos os jogadores simultaneamente..."

# Abrir 4 terminais simultaneamente
gnome-terminal --tab --title="Jogador 0" -- bash -c "go run main.go 0; exec bash" &
sleep 0.5
gnome-terminal --tab --title="Jogador 1" -- bash -c "go run main.go 1; exec bash" &
sleep 0.5
gnome-terminal --tab --title="Jogador 2" -- bash -c "go run main.go 2; exec bash" &
sleep 0.5
gnome-terminal --tab --title="Jogador 3" -- bash -c "go run main.go 3; exec bash" &

echo "Todos os jogadores iniciados!"