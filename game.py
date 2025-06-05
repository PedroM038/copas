import socket
import threading
import random
import time
import sys
import signal
import json

# LOGICA DE REDE
nodes = [
    ('localhost', 5000),
    ('localhost', 5001),
    ('localhost', 5002),
    ('localhost', 5003),
]

total_nodes = len(nodes)
current_node_index = int(sys.argv[1]) if len(sys.argv) > 1 else 0
next_node_index = (current_node_index + 1) % total_nodes

# Variáveis globais
game_over = False
game_winner = None
current_round = 0
current_trick = 0
current_trick_cards = []
player_hand = []
players_scores = [0, 0, 0, 0]
token = False
hearts_broken = False
trick_starter = None
current_trick_suit = None
first_trick = True
game_started = False
connected_players = set()
all_hands = []  # Para o host distribuir as cartas

# Socket DGRAM (UDP)
sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
sock.bind(nodes[current_node_index])

def send_message(message, target_node):
    target_address = nodes[target_node]
    sock.sendto(message.encode(), target_address)

def send_to_all(message):
    """Envia mensagem para todos os nós do anel"""
    for i in range(total_nodes):
        if i != current_node_index:
            send_message(message, i)

def receive_message():
    while not game_over:
        try:
            data, addr = sock.recvfrom(1024)
            message = data.decode()
            handle_message(message, addr)
        except socket.error as e:
            if not game_over:
                print(f"Erro ao receber mensagem: {e}")

threading.Thread(target=receive_message, daemon=True).start()

def pass_token():
    global token
    if token:
        send_message("TOKEN", next_node_index)
        token = False
        print(f"Token passed to node {next_node_index}")

# PROCESSAMENTO DE MENSAGENS
def handle_message(message, addr):
    global token, game_over, current_trick_cards, players_scores, game_started, connected_players
    
    try:
        data = json.loads(message)
        if data["type"] == "CONNECT":
            process_connect_message(data)
        elif data["type"] == "START_GAME":
            process_start_game_message(data)
        elif data["type"] == "TOKEN":
            process_token_message(data)
        elif data["type"] == "GAME":
            process_game_message(data)
        elif data["type"] == "SCORES":
            process_scores_message(data)
    except json.JSONDecodeError:
        if message == "TOKEN":
            token = True
            print(f"Received token from {addr}")

def process_connect_message(data):
    global connected_players, all_hands
    
    if current_node_index == 0:  # Host
        player_id = data["player"]
        connected_players.add(player_id)
        print(f"Player {player_id} connected. Total: {len(connected_players) + 1}/4")
        
        # Quando todos estiverem conectados, inicia o jogo
        if len(connected_players) == 3:  # 3 outros + host = 4 total
            print("All players connected! Starting game...")
            time.sleep(1)  # Pequena pausa para garantir sincronização
            start_game_as_host()

def process_start_game_message(data):
    global player_hand, game_started, token
    
    player_hand = data["hands"][current_node_index]
    game_started = True
    
    print(f"Game started! Your hand: {player_hand}")
    
    # O jogador com 2♣ recebe o token
    if "2♣" in player_hand:
        token = True
        print("You have 2♣, you start!")

def start_game_as_host():
    global all_hands, game_started, token, player_hand
    
    # Gera e distribui as cartas
    all_hands = deal_cards()
    player_hand = all_hands[0]  # Host pega a primeira mão
    
    # Envia as cartas para todos os jogadores
    start_message = {
        "type": "START_GAME",
        "hands": all_hands
    }
    send_to_all(json.dumps(start_message))
    
    game_started = True
    print(f"Game started! Your hand: {player_hand}")
    
    # O jogador com 2♣ recebe o token
    if "2♣" in player_hand:
        token = True
        print("You have 2♣, you start!")

def announce_connection():
    """Anuncia conexão para o host"""
    if current_node_index != 0:
        connect_message = {
            "type": "CONNECT",
            "player": current_node_index
        }
        send_message(json.dumps(connect_message), 0)  # Envia para o host

def process_token_message(data):
    global token
    token = True
    print("Received token")

def process_game_message(data):
    global current_trick_cards, hearts_broken, first_trick, current_trick
    
    if data["action"] == "PLAY":
        current_trick_cards.append({
            "card": data["card"],
            "player": data["player"]
        })
        print(f"Player {data['player']} played {data['card']}")
        
        if get_card_suit(data["card"]) == '♥':
            hearts_broken = True
        
        if len(current_trick_cards) == 4:
            winner_player = get_trick_winner(current_trick_cards)
            points = calculate_trick_points(current_trick_cards)
            
            # Atualiza pontuação local
            players_scores[winner_player] += points
            
            print(f"Player {winner_player} won the trick with {points} points")
            print(f"Current scores: {players_scores}")
            
            # Envia pontuação atualizada para todos
            scores_message = {
                "type": "SCORES",
                "scores": players_scores,
                "trick_winner": winner_player
            }
            send_to_all(json.dumps(scores_message))
            
            current_trick_cards = []
            first_trick = False
            current_trick += 1
            
            # Verifica fim de jogo
            if current_trick >= 13:
                check_game_end()

def process_scores_message(data):
    global players_scores
    players_scores = data["scores"]
    print(f"Updated scores: {players_scores}")

def create_deck():
    suits = ['♠', '♥', '♣', '♦']
    values = ['2', '3', '4', '5', '6', '7', '8', '9', '10', 'J', 'Q', 'K', 'A']
    deck = [f"{v}{s}" for s in suits for v in values]
    random.shuffle(deck)
    return deck

def deal_cards():
    deck = create_deck()
    hands = [deck[i:i + 13] for i in range(0, 52, 13)]
    return hands

def get_card_value(card):
    value = card[:-1]
    values = {'2': 2, '3': 3, '4': 4, '5': 5, '6': 6, '7': 7, '8': 8, 
              '9': 9, '10': 10, 'J': 11, 'Q': 12, 'K': 13, 'A': 14}
    return values[value]

def get_card_suit(card):
    return card[-1]

def is_valid_play(card, trick_cards, player_hand):
    global hearts_broken, first_trick
    
    # Primeira jogada de todas deve ser 2♣
    if first_trick and len(trick_cards) == 0:
        return card == "2♣"
    
    # Na primeira rodada, não pode jogar ♥ ou Q♠
    if first_trick:
        if get_card_suit(card) == '♥' or card == 'Q♠':
            # Só pode se não tiver outra opção
            other_cards = [c for c in player_hand if get_card_suit(c) != '♥' and c != 'Q♠']
            if len(trick_cards) == 0:
                return len(other_cards) == 0
            else:
                lead_suit = get_card_suit(trick_cards[0]["card"])
                same_suit_cards = [c for c in other_cards if get_card_suit(c) == lead_suit]
                return len(same_suit_cards) == 0
    
    # Se é o primeiro a jogar na rodada
    if len(trick_cards) == 0:
        # Copas não pode ser jogado até ser quebrado (exceto se só tiver copas)
        if get_card_suit(card) == '♥' and not hearts_broken:
            non_hearts = [c for c in player_hand if get_card_suit(c) != '♥']
            return len(non_hearts) == 0
        return True
    
    # Deve seguir o naipe se tiver
    lead_suit = get_card_suit(trick_cards[0]["card"])
    same_suit_cards = [c for c in player_hand if get_card_suit(c) == lead_suit]
    if same_suit_cards:
        return get_card_suit(card) == lead_suit
    
    return True

def get_card_points(card):
    if get_card_suit(card) == '♥':
        return 1
    elif card == 'Q♠':
        return 13
    return 0

def calculate_trick_points(trick_cards):
    points = 0
    for card_info in trick_cards:
        points += get_card_points(card_info["card"])
    return points

def get_trick_winner(trick_cards):
    lead_suit = get_card_suit(trick_cards[0]["card"])
    highest_value = -1
    winner_player = trick_cards[0]["player"]
    
    for card_info in trick_cards:
        card = card_info["card"]
        player = card_info["player"]
        if get_card_suit(card) == lead_suit:
            value = get_card_value(card)
            if value > highest_value:
                highest_value = value
                winner_player = player
    
    return winner_player

def play_card(card):
    global player_hand, hearts_broken
    
    if card in player_hand:
        player_hand.remove(card)
        if get_card_suit(card) == '♥':
            hearts_broken = True
        
        message = {
            "type": "GAME",
            "action": "PLAY",
            "card": card,
            "player": current_node_index
        }
        send_to_all(json.dumps(message))

def check_game_end():
    global game_over, game_winner
    
    max_score = max(players_scores)
    if max_score >= 100:
        game_over = True
        min_score = min(players_scores)
        game_winner = players_scores.index(min_score)
        print(f"\nGame Over! Player {game_winner} wins with {min_score} points!")
        print(f"Final scores: {players_scores}")

def initialize_connection():
    global current_round, game_over, hearts_broken, first_trick
    
    current_round = 0
    game_over = False
    hearts_broken = False
    first_trick = True
    
    if current_node_index == 0:
        print("Player 0 (Host) - Waiting for other players to connect...")
        connected_players.add(0)  # Host está conectado
    else:
        print(f"Player {current_node_index} - Connecting to game...")
        announce_connection()
        print("Waiting for game to start...")

def main():
    initialize_connection()
    
    # Aguarda o jogo começar
    while not game_started and not game_over:
        time.sleep(0.5)
    
    # Loop principal do jogo
    while not game_over:
        if token:
            print(f"\nSua mão: {player_hand}")
            valid_cards = [c for c in player_hand if is_valid_play(c, current_trick_cards, player_hand)]
            
            if valid_cards:
                print(f"Cartas válidas: {valid_cards}")
                print(f"Current trick: {[card_info['card'] for card_info in current_trick_cards]}")
                card = input("Escolha uma carta para jogar: ").strip()
                if card in valid_cards:
                    play_card(card)
                    pass_token()
                else:
                    print("Carta inválida! Tente novamente.")
            else:
                print("Nenhuma carta válida disponível!")
        
        time.sleep(0.1)

def signal_handler(sig, frame):
    global game_over
    print("\nShutting down...")
    game_over = True
    sock.close()
    sys.exit(0)

signal.signal(signal.SIGINT, signal_handler)

if __name__ == "__main__":
    main()