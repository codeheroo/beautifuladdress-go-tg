package main

import (
	"crypto/ecdsa"
	"fmt"
	"log"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/tucnak/telebot"
	"github.com/tyler-smith/go-bip32"
	"github.com/tyler-smith/go-bip39"
)

// Function to validate the prefix or suffix
func isValidPrefix(input string) bool {
	if !strings.HasPrefix(input, "0x") {
		return false
	}
	if len(input) > 7 {
		return false
	}
	validHex := regexp.MustCompile(`^0x[0-9a-fA-F]*$`)
	return validHex.MatchString(input)
}

func isValidSuffix(input string) bool {

	if len(input) > 5 {
		return false
	}
	validHex := regexp.MustCompile(`^[0-9a-fA-F]*$`)
	return validHex.MatchString(input)
}

func isValidPrefixSuffix(prefix string, suffix string) bool {
	if !strings.HasPrefix(prefix, "0x") {
		return false
	}
	if (len(prefix) + len(suffix)) > 7 {
		return false
	}
	validPrefix := regexp.MustCompile(`^0x[0-9a-fA-F]*$`)
	validSuffix := regexp.MustCompile(`^[0-9a-fA-F]*$`)
	return validPrefix.MatchString(prefix) && validSuffix.MatchString(suffix)
}

// Function to generate mnemonic and address by prefix, suffix, or both
func generateMnemonicAndAddress(prefix, suffix string, results chan<- string, wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		// Generate a random 256-bit seed
		entropy, err := bip39.NewEntropy(256)
		if err != nil {
			log.Fatal(err)
		}

		// Generate a mnemonic from the entropy
		mnemonic, err := bip39.NewMnemonic(entropy)
		if err != nil {
			log.Fatal(err)
		}

		// Generate a seed from the mnemonic
		seed := bip39.NewSeed(mnemonic, "")

		// Generate a master key from the seed
		masterKey, err := bip32.NewMasterKey(seed)
		if err != nil {
			log.Fatal(err)
		}

		// Derive the Ethereum address from the master key
		privateKeyECDSA, err := crypto.ToECDSA(masterKey.Key)
		if err != nil {
			log.Fatal(err)
		}

		publicKey := privateKeyECDSA.Public()
		publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
		if !ok {
			log.Fatal("error casting public key to ECDSA")
		}

		address := crypto.PubkeyToAddress(*publicKeyECDSA).Hex()

		if strings.HasPrefix(address, prefix) && strings.HasSuffix(address, suffix) {
			result := fmt.Sprintf("📝 Mnemonic: %s\n🏷️ Address: %s", mnemonic, address)
			results <- result
			return
		}
	}
}

func main() {
	// Replace "YOUR_TELEGRAM_BOT_TOKEN" with your actual Telegram bot token
	bot, err := telebot.NewBot(telebot.Settings{
		Token:  "",
		Poller: &telebot.LongPoller{Timeout: 10 * time.Second},
	})
	if err != nil {
		log.Fatal(err)
	}

	// Handle /start command
	bot.Handle("/start", func(m *telebot.Message) {
		intro := `👋 Welcome to the Ethereum Address Generator Bot!
I can help you generate an Ethereum address that starts with a specific prefix, ends with a specific suffix, or both.

🔹 To use me, send one of the following commands:
  - /prefix <your_prefix> to generate an address starting with your prefix
  - /suffix <your_suffix> to generate an address ending with your suffix
  - /both <your_prefix> <your_suffix> to generate an address starting and ending with your prefix and suffix

The prefix and suffix should start with "0x", contain only numbers and letters a-f or A-F, and be no longer than 7 symbols.

🚀 Try sending me a command now!`
		bot.Send(m.Sender, intro)
	})

	// Handle /prefix command
	bot.Handle("/prefix", func(m *telebot.Message) {
		args := strings.Split(m.Payload, " ")
		if len(args) != 1 || !isValidPrefix(args[0]) {
			bot.Send(m.Sender, "❌ Invalid prefix. It should start with '0x', contain only numbers and letters a-f or A-F, and be no longer than 7 symbols.")
			return
		}

		prefix := args[0]
		results := make(chan string)
		var wg sync.WaitGroup

		numThreads := 4 // Adjust the number of threads as needed

		for i := 0; i < numThreads; i++ {
			wg.Add(1)
			go generateMnemonicAndAddress(prefix, "", results, &wg)
		}

		go func() {
			wg.Wait()
			close(results)
		}()

		result := <-results
		bot.Send(m.Sender, result)
	})

	// Handle /suffix command
	bot.Handle("/suffix", func(m *telebot.Message) {
		args := strings.Split(m.Payload, " ")
		if len(args) != 1 || !isValidSuffix(args[0]) {
			bot.Send(m.Sender, "❌ Invalid suffix. It should contain only numbers and letters a-f or A-F, and be no longer than 5 symbols.")
			return
		}

		suffix := args[0]
		results := make(chan string)
		var wg sync.WaitGroup

		numThreads := 4 // Adjust the number of threads as needed

		for i := 0; i < numThreads; i++ {
			wg.Add(1)
			go generateMnemonicAndAddress("", suffix, results, &wg)
		}

		go func() {
			wg.Wait()
			close(results)
		}()

		result := <-results
		bot.Send(m.Sender, result)
	})

	// Handle /both command
	bot.Handle("/both", func(m *telebot.Message) {
		args := strings.Split(m.Payload, " ")
		if len(args) != 2 || !isValidPrefixSuffix(args[0], args[1]) {
			bot.Send(m.Sender, "❌ Invalid prefix or suffix. Prefix should start with '0x'. They both should contain only numbers and letters a-f or A-F, and be no longer than 7 symbols together.")
			return
		}

		prefix := args[0]
		suffix := args[1]
		results := make(chan string)
		var wg sync.WaitGroup

		numThreads := 1000 // Adjust the number of threads as needed

		for i := 0; i < numThreads; i++ {
			wg.Add(1)
			go generateMnemonicAndAddress(prefix, suffix, results, &wg)
		}

		go func() {
			wg.Wait()
			close(results)
		}()

		result := <-results
		bot.Send(m.Sender, result)
	})

	bot.Start()
}
