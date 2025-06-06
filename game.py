import threading
import random
import time
import sys
import signal
import json
import socket

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
current_trick = 1
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
    
    # Log da mensagem enviada
    try:
        msg_data = json.loads(message)
        log_message("SEND", msg_data.get("type", "UNKNOWN"), target_node, msg_data)
    except:
        log_message("SEND", "TOKEN", target_node, message)

def send_to_all(message):
    """Envia mensagem para todos os nós do anel"""
    try:
        msg_data = json.loads(message)
        log_message("BROADCAST", msg_data.get("type", "UNKNOWN"), None, msg_data)
    except:
        log_message("BROADCAST", "UNKNOWN", None, message)
    
    for i in range(total_nodes):
        send_message(message, i)

def receive_message():
    while not game_over:
        try:
            data, addr = sock.recvfrom(1024)
            message = data.decode()
            handle_message(message, addr)
        except socket.error as e:
            if not game_over:
                print(f"⚠️ Erro ao receber mensagem: {e}")

threading.Thread(target=receive_message, daemon=True).start()

def pass_token(node_index):
    global token
    if token:
        send_message("TOKEN", node_index)
        token = False
        print(f"🎯 Token passado para Player {node_index}")


def log_message(action, message_type, target=None, data=None):
    """Log padronizado para mensagens enviadas/recebidas"""
    timestamp = time.strftime("%H:%M:%S")
    if action == "SEND":
        print(f"[{timestamp}] 📤 SEND to Player {target}: {message_type} - {data}")
    elif action == "RECV":
        print(f"[{timestamp}] 📥 RECV from {target}: {message_type} - {data}")
    elif action == "BROADCAST":
        print(f"[{timestamp}] 📢 BROADCAST: {message_type} - {data}")

# PROCESSAMENTO DE MENSAGENS

def handle_message(message, addr):
    global token
    
    # Se for apenas o token, processa diretamente
    if message == "TOKEN":
        token = True
        log_message("RECV", "TOKEN", addr, "Token recebido")
        print(f"🎯 Token recebido! É sua vez de jogar.")
        return
    
    # Tenta processar como JSON
    try:
        data = json.loads(message)
        msg_type = data.get("type")
        
        log_message("RECV", msg_type, addr, data)
        
        if msg_type == "CONNECT":
            process_connect_message(data)
        elif msg_type == "START_GAME":
            process_start_game_message(data)
        elif msg_type == "GAME":
            process_game_message(data)
        elif msg_type == "END_TRICK":
            process_end_trick_message(data)
        elif msg_type == "SCORES":
            process_scores_message(data)
        elif msg_type == "NEW_HAND":
            process_new_hand_message(data)
        elif msg_type == "GAME_END":
            process_game_end_message(data)
        else:
            print(f"⚠️ Tipo de mensagem desconhecido: {msg_type}")
    
    except json.JSONDecodeError:
        print(f"⚠️ Erro ao decodificar JSON: {message}")

def process_connect_message(data):
    global connected_players
    
    player_id = data.get("player")
    
    # Apenas o host (player 0) processa conexões
    if current_node_index == 0:
        connected_players.add(player_id)
        print(f"🔗 Player {player_id} conectado! ({len(connected_players)}/4)")
        
        # Se todos os 4 jogadores estão conectados, inicia o jogo
        if len(connected_players) == 4:
            print("🎉 Todos os jogadores conectados! Iniciando jogo...")
            start_game_as_host()

def process_start_game_message(data):
    global game_started, player_hand, token
    
    hands = data.get("hands", [])
    
    if current_node_index < len(hands):
        player_hand = hands[current_node_index]
        game_started = True
                
        # Verifica se este jogador tem o 2♣ e deve começar
        if "2♣" in player_hand:
            send_message("TOKEN", current_node_index)
        else:
            token = False
            print(f"🔄 Player {current_node_index} não tem o 2♣, esperando o próximo jogador.")

def process_game_message(data):
    global current_trick_cards, current_trick_suit, trick_starter
    
    action = data.get("action")
    
    if action == "PLAY":
        card = data.get("card")
        player = data.get("player")
        
        # Adiciona a carta jogada às cartas da rodada atual
        current_trick_cards.append({
            "card": card,
            "player": player
        })
        
        #printa quantidade de cartas jogadas no trick atual
        print(f"📋 Cartas jogadas no trick atual: {len(current_trick_cards)}")

        # Se é a primeira carta da rodada, define o naipe da rodada e quem começou
        if len(current_trick_cards) == 1:
            current_trick_suit = get_card_suit(card)
            trick_starter = player
        
        print(f"🃏 Player {player} jogou: {card}")
        print(f"📋 Cartas na mesa: {[c['card'] for c in current_trick_cards]}")
        
        # Se todos os 4 jogadores jogaram, termina a rodada
        if len(current_trick_cards) == 4:
            end_trick()

def process_end_trick_message(data):
    global current_trick_cards, current_trick, first_trick, players_scores, token
    
    winner = data.get("winner")
    points = data.get("points")
    scores = data.get("scores")
    
    print(f"\n🏆 Player {winner} ganhou a rodada!")
    print(f"📊 Pontos da rodada: {points}")
    
    # Atualiza pontuações
    if scores:
        players_scores = scores.copy()
        print(f"📈 Pontuações atuais: {players_scores}")
    
    # Limpa as cartas da mesa
    current_trick_cards = []
    first_trick = False
    
    # Verifica se completou uma mão (13 rodadas)
    if current_trick >= 13:
        check_game_end()

def process_scores_message(data):
    global players_scores
    
    scores = data.get("scores")
    if scores:
        players_scores = scores.copy()
        print(f"📊 Pontuações atualizadas: {players_scores}")

def process_new_hand_message(data):
    global player_hand, current_trick, first_trick, hearts_broken, token
    
    hands = data.get("hands", [])
    
    if current_node_index < len(hands):
        player_hand = hands[current_node_index]
        current_trick = 0
        first_trick = True
        hearts_broken = False
        token = False
        
        print(f"🆕 Nova mão iniciada! Suas cartas: {sorted(player_hand, key=lambda x: (get_card_suit(x), get_card_value(x)))}")
        
        # Verifica se este jogador tem o 2♣ e deve começar
        if "2♣" in player_hand:
            send_message("TOKEN", current_node_index)
        else:
            token = False
            print(f"🔄 Player {current_node_index} não tem o 2♣, esperando o próximo jogador.")

def process_game_end_message(data):
    global game_over, game_winner
    
    winner = data.get("winner")
    final_scores = data.get("final_scores")
    
    game_over = True
    game_winner = winner
    
    print(f"\n🎉 JOGO TERMINADO!")
    print(f"🏆 Player {winner} venceu com {final_scores[winner]} pontos!")
    print(f"📊 Pontuações finais: {final_scores}")
    
    # Encerra o programa após alguns segundos
    time.sleep(3)
    sock.close()
    sys.exit(0)
    
def end_trick():
    global current_trick_cards, players_scores, current_trick
    
    # Calcula quem ganhou a rodada
    winner = get_trick_winner(current_trick_cards)
    
    # Calcula pontos da rodada
    points = calculate_trick_points(current_trick_cards)
    
    # Adiciona pontos ao vencedor
    players_scores[winner] += points
    
    print(f"\n🏆 Rodada {current_trick} finalizada!")
    print(f"📋 Cartas jogadas: {[card_info['card'] for card_info in current_trick_cards]}")
    print(f"🎯 Vencedor: Player {winner}")
    print(f"💔 Pontos: {points}")
    
    current_trick += 1
    
    # Envia resultado para todos os jogadores
    end_trick_message = {
        "type": "END_TRICK",
        "winner": winner,
        "points": points,
        "scores": players_scores.copy(),
        "trick": current_trick + 1
    }
    send_to_all(json.dumps(end_trick_message))

# LOGICA DO JOGO

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

def get_card_number_value(card):
    """Converte carta para valor numérico para display"""
    value = card[:-1]
    if value == 'J':
        return 11
    elif value == 'Q':
        return 12
    elif value == 'K':
        return 13
    elif value == 'A':
        return 14
    else:
        return int(value)

def find_card_by_number(number, valid_cards):
    """Encontra carta na mão pelo número digitado"""
    for card in valid_cards:
        if get_card_number_value(card) == number:
            return card
    return None

def display_cards_with_numbers(cards):
    """Exibe cartas com seus números correspondentes"""
    print("\n🃏 Suas cartas:")
    for i, card in enumerate(sorted(cards, key=lambda x: (get_card_suit(x), get_card_value(x)))):
        number = get_card_number_value(card)
        print(f"   {number}: {card}")

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
        print(f"🃏 Você jogou: {card}")
        pass_token(next_node_index)
    else:        
        print("❌ Você não tem essa carta na mão!")
        return

def check_game_end():
    global game_over, game_winner
    
    max_score = max(players_scores)
    if max_score >= 100:
        game_over = True
        min_score = min(players_scores)
        game_winner = players_scores.index(min_score)
        
        # Envia mensagem de fim de jogo para todos
        game_end_message = {
            "type": "GAME_END",
            "winner": game_winner,
            "final_scores": players_scores.copy()
        }
        send_to_all(json.dumps(game_end_message))
        
        print(f"\n🎉 JOGO TERMINADO!")
        print(f"🏆 Player {game_winner} venceu com {min_score} pontos!")
        print(f"📊 Pontuações finais: {players_scores}")
    else:
        # Se o jogo não acabou mas completou uma mão (13 tricks), inicia nova mão
        print(f"\n🔄 Mão completada! Pontuações atuais: {players_scores}")
        if max_score < 100:
            start_new_hand()

def start_new_hand():
    global current_trick, first_trick, hearts_broken, player_hand, all_hands, token
    
    print("🆕 Iniciando nova mão...")
    current_trick = 0
    first_trick = True
    hearts_broken = False
    
    if current_node_index == 0:  # Host redistribui as cartas
        all_hands = deal_cards()
        player_hand = all_hands[0]
        
        start_message = {
            "type": "NEW_HAND",
            "hands": all_hands
        }
        send_to_all(json.dumps(start_message))
    
    if "2♣" in player_hand:
        send_message("TOKEN", current_node_index)
    else:
        token = False
        print(f"🔄 Player {current_node_index} não tem o 2♣, esperando o próximo jogador.")

# Conexão e início do jogo
def start_game_as_host():
    global all_hands, game_started, token, player_hand
    
    print("🎲 Host distribuindo cartas...")
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
    print(f"🎮 Jogo iniciado! Suas cartas: {player_hand}")
    
    # O jogador com 2♣ recebe o token
    if "2♣" in player_hand:
        send_message("TOKEN", current_node_index)
    else:
        token = False
        print(f"🔄 Player {current_node_index} não tem o 2♣, esperando o próximo jogador.")

def initialize_connection():
    global current_round, game_over, hearts_broken, first_trick
    
    current_round = 0
    game_over = False
    hearts_broken = False
    first_trick = True
    
    if current_node_index == 0:
        print(f"🎮 Player {current_node_index} (Host) - Aguardando outros jogadores...")
        connected_players.add(0)  # Host está conectado
    else:
        print(f"🎮 Player {current_node_index} - Conectando ao jogo...")
        announce_connection()
        print("⏳ Aguardando início do jogo...")

def announce_connection():
    """Anuncia conexão para o host"""
    if current_node_index != 0:
        connect_message = {
            "type": "CONNECT",
            "player": current_node_index
        }
        send_message(json.dumps(connect_message), 0)  # Envia para o host
        print("📡 Conexão anunciada para o host")


# main loop do jogo
def main():
    print(f"🚀 Iniciando Player {current_node_index}")
    initialize_connection()
    
    # Aguarda o jogo começar
    while not game_started and not game_over:
        time.sleep(0.5)
    
    # Loop principal do jogo
    while not game_over:
        if token:
            print(f"\n{'='*50}")
            print(f"🎯 SUA VEZ! (Player {current_node_index})")
            
            display_cards_with_numbers(player_hand)
            
            valid_cards = [c for c in player_hand if is_valid_play(c, current_trick_cards, player_hand)]
            
            if valid_cards:
                print(f"\n✅ Cartas válidas para jogar:")
                for card in sorted(valid_cards, key=lambda x: (get_card_suit(x), get_card_value(x))):
                    number = get_card_number_value(card)
                    print(f"   {number}: {card}")
                
                if current_trick_cards:
                    print(f"\n🃏 Cartas na mesa: {[card_info['card'] for card_info in current_trick_cards]}")
                
                try:
                    choice = int(input("\n🎯 Digite o número da carta para jogar: "))
                    selected_card = find_card_by_number(choice, valid_cards)
                    
                    if selected_card:
                        play_card(selected_card)
                    else:
                        print("❌ Número inválido! Tente novamente.")
                except ValueError:
                    print("❌ Digite apenas números!")
            else:
                print("⚠️ Nenhuma carta válida disponível!")
        
        time.sleep(0.1)

def signal_handler(sig, frame):
    global game_over
    print("\n🛑 Encerrando...")
    game_over = True
    sock.close()
    sys.exit(0)

signal.signal(signal.SIGINT, signal_handler)

if __name__ == "__main__":
    main()