package network

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

// Níveis de log
const (
	LOG_DEBUG = "DEBUG"
	LOG_INFO  = "INFO"
	LOG_WARN  = "WARN"
	LOG_ERROR = "ERROR"
	LOG_FATAL = "FATAL"
)

// Cores para output no terminal
const (
	COLOR_RESET  = "\033[0m"
	COLOR_RED    = "\033[31m"
	COLOR_GREEN  = "\033[32m"
	COLOR_YELLOW = "\033[33m"
	COLOR_BLUE   = "\033[34m"
	COLOR_PURPLE = "\033[35m"
	COLOR_CYAN   = "\033[36m"
	COLOR_WHITE  = "\033[37m"
)

// Logger para rede
type NetworkLogger struct {
	nodeID    int
	enabled   bool
	level     string
	logger    *log.Logger
	useColors bool
	logFile   *os.File
}

// Cria um novo logger de rede
func NewNetworkLogger(nodeID int, useColors bool) *NetworkLogger {
	logger := &NetworkLogger{
		nodeID:    nodeID,
		enabled:   true,
		level:     LOG_INFO,
		useColors: useColors,
		logger:    log.New(os.Stdout, "", 0),
	}

	// Tenta criar arquivo de log
	filename := fmt.Sprintf("node_%d.log", nodeID)
	if file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666); err == nil {
		logger.logFile = file
	}

	return logger
}

// Define o nível de log
func (nl *NetworkLogger) SetLevel(level string) {
	nl.level = level
}

// Habilita/desabilita logs
func (nl *NetworkLogger) SetEnabled(enabled bool) {
	nl.enabled = enabled
}

// Método principal de log
func (nl *NetworkLogger) log(level, category, message string, args ...interface{}) {
	if !nl.enabled || !nl.shouldLog(level) {
		return
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	formattedMsg := fmt.Sprintf(message, args...)

	// Formato: [TIMESTAMP] [NODE_ID] [LEVEL] [CATEGORY] MESSAGE
	logLine := fmt.Sprintf("[%s] [NODE_%d] [%s] [%s] %s",
		timestamp, nl.nodeID, level, category, formattedMsg)

	// Log colorido para terminal
	if nl.useColors {
		coloredLine := nl.colorize(level, logLine)
		nl.logger.Println(coloredLine)
	} else {
		nl.logger.Println(logLine)
	}

	// Log para arquivo (sem cores)
	if nl.logFile != nil {
		nl.logFile.WriteString(logLine + "\n")
		nl.logFile.Sync()
	}
}

// Verifica se deve logar baseado no nível
func (nl *NetworkLogger) shouldLog(level string) bool {
	levels := map[string]int{
		LOG_DEBUG: 0,
		LOG_INFO:  1,
		LOG_WARN:  2,
		LOG_ERROR: 3,
		LOG_FATAL: 4,
	}

	currentLevel, exists := levels[nl.level]
	if !exists {
		return true
	}

	msgLevel, exists := levels[level]
	if !exists {
		return true
	}

	return msgLevel >= currentLevel
}

// Adiciona cores ao log
func (nl *NetworkLogger) colorize(level, message string) string {
	switch level {
	case LOG_DEBUG:
		return COLOR_CYAN + message + COLOR_RESET
	case LOG_INFO:
		return COLOR_GREEN + message + COLOR_RESET
	case LOG_WARN:
		return COLOR_YELLOW + message + COLOR_RESET
	case LOG_ERROR:
		return COLOR_RED + message + COLOR_RESET
	case LOG_FATAL:
		return COLOR_PURPLE + message + COLOR_RESET
	default:
		return message
	}
}

// Logs específicos da rede

// Log de conexão
func (nl *NetworkLogger) LogConnection(address string, status string) {
	nl.log(LOG_INFO, "CONNECTION", "Conexão %s: %s", address, status)
}

// Log de mensagem enviada
func (nl *NetworkLogger) LogMessageSent(msg *Message) {
	nl.log(LOG_INFO, "MESSAGE_SENT", "Enviada: %s de %d para %d (ID: %s, Hops: %d)",
		msg.Type, msg.From, msg.To, msg.MessageID, msg.Hops)
}

// Log de mensagem recebida
func (nl *NetworkLogger) LogMessageReceived(msg *Message) {
	nl.log(LOG_INFO, "MESSAGE_RECEIVED", "Recebida: %s de %d para %d (ID: %s, Hops: %d)",
		msg.Type, msg.From, msg.To, msg.MessageID, msg.Hops)
}

// Log de mensagem processada
func (nl *NetworkLogger) LogMessageProcessed(msg *Message, action string) {
	nl.log(LOG_INFO, "MESSAGE_PROCESSED", "Processada: %s - %s (ID: %s)",
		msg.Type, action, msg.MessageID)
}

// Log de mensagem encaminhada
func (nl *NetworkLogger) LogMessageForwarded(msg *Message) {
	nl.log(LOG_DEBUG, "MESSAGE_FORWARDED", "Encaminhada: %s (ID: %s, Hops: %d)",
		msg.Type, msg.MessageID, msg.Hops)
}

// Log de token
func (nl *NetworkLogger) LogTokenReceived(token *Token) {
	nl.log(LOG_INFO, "TOKEN", "Token recebido (ID: %s, Sequence: %d)",
		token.ID, token.Sequence)
}

func (nl *NetworkLogger) LogTokenPassed(token *Token, nextNode int) {
	nl.log(LOG_INFO, "TOKEN", "Token passado para nó %d (ID: %s, Sequence: %d)",
		nextNode, token.ID, token.Sequence)
}

func (nl *NetworkLogger) LogTokenWaiting() {
	nl.log(LOG_DEBUG, "TOKEN", "Aguardando token...")
}

// Log de erros
func (nl *NetworkLogger) LogError(operation string, err error) {
	nl.log(LOG_ERROR, "ERROR", "Erro em %s: %v", operation, err)
}

func (nl *NetworkLogger) LogWarning(operation string, message string) {
	nl.log(LOG_WARN, "WARNING", "%s: %s", operation, message)
}

// Log de estatísticas
func (nl *NetworkLogger) LogStatistics(stats map[string]interface{}) {
	var statLines []string
	for key, value := range stats {
		statLines = append(statLines, fmt.Sprintf("%s: %v", key, value))
	}
	nl.log(LOG_INFO, "STATISTICS", "Estatísticas: %s", strings.Join(statLines, ", "))
}

// Log de estado da rede
func (nl *NetworkLogger) LogNetworkState(state string, details string) {
	nl.log(LOG_INFO, "NETWORK_STATE", "Estado: %s - %s", state, details)
}

// Log de timeout
func (nl *NetworkLogger) LogTimeout(operation string, duration time.Duration) {
	nl.log(LOG_WARN, "TIMEOUT", "Timeout em %s após %v", operation, duration)
}

// Log de início/fim de operações
func (nl *NetworkLogger) LogOperationStart(operation string) {
	nl.log(LOG_DEBUG, "OPERATION", "Iniciando: %s", operation)
}

func (nl *NetworkLogger) LogOperationEnd(operation string, success bool) {
	status := "SUCESSO"
	level := LOG_DEBUG
	if !success {
		status = "FALHA"
		level = LOG_WARN
	}
	nl.log(level, "OPERATION", "Finalizando: %s - %s", operation, status)
}

// Log de debug para desenvolvimento
func (nl *NetworkLogger) Debug(message string, args ...interface{}) {
	nl.log(LOG_DEBUG, "DEBUG", message, args...)
}

func (nl *NetworkLogger) Info(message string, args ...interface{}) {
	nl.log(LOG_INFO, "INFO", message, args...)
}

func (nl *NetworkLogger) Warn(message string, args ...interface{}) {
	nl.log(LOG_WARN, "WARN", message, args...)
}

func (nl *NetworkLogger) Error(message string, args ...interface{}) {
	nl.log(LOG_ERROR, "ERROR", message, args...)
}

func (nl *NetworkLogger) Fatal(message string, args ...interface{}) {
	nl.log(LOG_FATAL, "FATAL", message, args...)
	if nl.logFile != nil {
		nl.logFile.Close()
	}
	os.Exit(1)
}

// Fecha o logger
func (nl *NetworkLogger) Close() {
	if nl.logFile != nil {
		nl.logFile.Close()
	}
}

// Log detalhado de mensagem
func (nl *NetworkLogger) LogMessageDetails(msg *Message, direction string) {
	details := fmt.Sprintf("Tipo: %s, De: %d, Para: %d, Hops: %d, Prioridade: %d, Timestamp: %d",
		msg.Type, msg.From, msg.To, msg.Hops, msg.Priority, msg.Timestamp)
	nl.log(LOG_DEBUG, "MESSAGE_DETAILS", "%s - %s (ID: %s)", direction, details, msg.MessageID)
}
