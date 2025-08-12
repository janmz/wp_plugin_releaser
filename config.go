package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"log"
	mathRand "math/rand"
	"os"
	"reflect"
	"strconv"
	"strings"

	"crypto/aes"      // AES Encryption
	"crypto/cipher"   // Cipher for GCM
	"encoding/base64" // Base64 Encoding
)

// PASSWORD_IS_SECURE is an Indicator that the password is only stored encrypted.
// All other values are interpreted as a new password and then encrypted.
const PASSWORD_IS_SECURE = "Hier neues Passwort eintragen"

var encryptionKey []byte
var initialized = false

/*
 * Password key initialization
 *
 * This involves initializing from the computer's hardware properties.
 * This makes the file unusable on another computer - this is an
 * additional security feature.
 *
 * For transferring files of the first version of this application, an old,
 * insecure key generation procedure can also be used.
 */
func config_init() {
	if !initialized {
		// Generate encryption key based on Hardware IS
		hardwareID, err := getHardwareID()
		if err != nil {
			//lint:ignore (ST1005) German error message requires capitalization
			log.Fatalf("Config: Hardware ID kann nicht bestimmt werden")
		}
		randGenSeeded := mathRand.NewSource(int64(hardwareID))
		encryptionKey = make([]byte, 32)
		for i := range encryptionKey {
			encryptionKey[i] = byte(randGenSeeded.Int63() >> 16 & 0xff)
		}
	}
	initialized = true
}

/*
 * Go through the structure and set the default values present
 * in the annotations
 */
func process_defaults(v reflect.Value) error {
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return nil
	}
	t := v.Type()
	// Iterate through all fields
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)
		if field.Type.Kind() == reflect.Struct {
			if err := process_defaults(fieldValue); err != nil {
				//lint:ignore ST1005 German error message requires capitalization
				return fmt.Errorf("Fehler beim Auslesen der default-Version der config-Datei: %v", err)
			}
		} else if field.Type.Kind() == reflect.Slice {
			for i := 0; i < fieldValue.Len(); i++ {
				if fieldValue.Index(i).Kind() == reflect.Struct {
					if err := process_defaults(fieldValue.Index(i)); err != nil {
						return err
					}
				}
			}
		} else {
			defaultValue, found := field.Tag.Lookup("default")
			if found {
				switch fieldValue.Kind() {
				case reflect.String:
					fieldValue.SetString(defaultValue)
				case reflect.Int, reflect.Int64:
					value, err := strconv.Atoi(defaultValue)
					if err != nil {
						//lint:ignore ST1005 German error message requires capitalization
						return fmt.Errorf("Fehler beim Auslesen der default-Version der config-Datei: %v", err)
					}
					fieldValue.SetInt(int64(value))
				case reflect.Bool:
					boolValue, err := strconv.ParseBool(defaultValue)
					if err != nil {
						//lint:ignore ST1005 German error message requires capitalization
						return fmt.Errorf("Fehler beim Auslesen der default-Version der config-Datei: %v", err)
					}
					fieldValue.SetBool(boolValue)
				default:
					//lint:ignore ST1005 German error message requires capitalization
					return fmt.Errorf("Unsupported type for default value: %v", fieldValue.Kind())
				}
			}
		}
	}
	return nil
}

/*
 * Check new content and update encrypted passwords and version as needed
 * If changes are made, the modified file will be written back at the end
 */
func process_checks(v reflect.Value, version int, changed *bool) error {
	if v.Kind() == reflect.Ptr {
		//fmt.Printf("Pointer\n")
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		//fmt.Printf("Keine Struktur sondern %s\n", v.Kind().String())
		return nil
	}
	t := v.Type()
	// Iterate through all fields
	for i := 0; i < t.NumField(); i++ {

		field := t.Field(i)
		fieldValue := v.Field(i)

		//fmt.Printf("Feld %d: %s(%s) = %v\n", i, field.Name, fieldValue.Kind().String(), fieldValue)

		// Process nested structures recursively
		if field.Type.Kind() == reflect.Struct {
			if err := process_checks(fieldValue, version, changed); err != nil {
				return err
			}
		} else if field.Type.Kind() == reflect.Slice {
			//fmt.Printf("Slice[0..%d]\n", fieldValue.Len()-1)
			for i := 0; i < fieldValue.Len(); i++ {
				//fmt.Printf("Slice-Element %d:\n", i)
				if fieldValue.Index(i).Kind() == reflect.Struct {
					if err := process_checks(fieldValue.Index(i), version, changed); err != nil {
						return err
					}
				} else {
					//fmt.Printf(" is '%v' (%s)\n", fieldValue.Index(i), fieldValue.Index(i).Kind().String())
				}
			}
		} else {
			// Version check
			if field.Name == "Version" {
				if fieldValue.Int() != int64(version) {
					fieldValue.SetInt(int64(version))
					//fmt.Printf(" neuer Wert %d\n", version)
					*changed = true
				}
			}
			// Password handling
			if strings.HasSuffix(field.Name, "SecurePassword") {
				pw_prefix := strings.TrimSuffix(field.Name, "SecurePassword")
				for j := 0; j < t.NumField(); j++ {
					if t.Field(j).Name == pw_prefix+"Password" {
						field2Value := v.Field(j)
						if field2Value.String() != PASSWORD_IS_SECURE {
							// Neues Passwort im Klartext gefunden
							// Neues Secure_Password wird berechnet
							password := encrypt(field2Value.String())
							fieldValue.SetString(password)
							field2Value.SetString(PASSWORD_IS_SECURE)
							//fmt.Printf(" neuer Wert %s\n", password)
							*changed = true
						}
						break
					}
				}
			}
		}
	}
	return nil
}

/*
 * Decrypt the encrypted passwords so that the encryption is transparent in the main program.
 */
func process_decode(v reflect.Value) error {
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return nil
	}
	t := v.Type()
	// Iterate through all fields
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)

		// Process recursively nested structures
		if field.Type.Kind() == reflect.Struct {
			if err := process_decode(fieldValue); err != nil {
				return err
			}
		} else if field.Type.Kind() == reflect.Slice {
			for i := 0; i < fieldValue.Len(); i++ {
				if fieldValue.Index(i).Kind() == reflect.Struct {
					if err := process_decode(fieldValue.Index(i)); err != nil {
						return err
					}
				}
			}
		} else {
			// Password processing
			if strings.HasSuffix(field.Name, "SecurePassword") {
				pw_prefix := strings.TrimSuffix(field.Name, "SecurePassword")
				for j := 0; j < t.NumField(); j++ {
					if t.Field(j).Name == pw_prefix+"Password" {
						field2Value := v.Field(j)
						password, err := decrypt(fieldValue.String())
						if err != nil {
							//lint:ignore ST1005 German error message requires capitalization
							return fmt.Errorf("Failed to decrypt %s password: %v", pw_prefix, err)
						}
						field2Value.SetString(password)
						break
					}
				}
			}
		}
	}
	return nil
}

/*
 * Reading a JSON file containing data for the config struct, where
 * passwords are encrypted and decrypted, and if the cleanConfig parameter
 * is set, a file with unencrypted passwords is written.
 */
func loadConfig(config interface{}, version int, path string, cleanConfig bool) error {

	var file []byte

	config_init()

	_, err := os.Stat(path)
	if !os.IsNotExist(err) {
		file, err = os.ReadFile(path)
		if err != nil {
			//lint:ignore ST1005 German error message requires capitalization
			return fmt.Errorf("Failed to read config file: %v", err)
		}
	} else {
		file = make([]byte, 0)
	}

	// Analyze config type
	configValue := reflect.ValueOf(config)
	if configValue.Kind() == reflect.Ptr {
		configValue = configValue.Elem()
	}
	if configValue.Kind() != reflect.Struct {
		return fmt.Errorf("config must be a pointer to a struct")
	}

	if err := process_defaults(configValue); err != nil {
		return fmt.Errorf("failed to set default config entries: %v", err)
	}

	if err := json.Unmarshal(file, config); err != nil {
		return fmt.Errorf("failed to parse config file: %v", err)
	}
	changed := false
	if err := process_checks(configValue, version, &changed); err != nil {
		return fmt.Errorf("failed to check config entries: %v", err)
	}
	if cleanConfig {
		/* Decrypt passwords before writing */
		if err := process_decode(configValue); err != nil {
			return fmt.Errorf("failed to decode passwords in config entries: %v", err)
		}
		changed = true
	}
	if changed {
		configJSON, err := json.MarshalIndent(config, "", "\t")
		if err != nil {
			return fmt.Errorf("failed to marshal config to JSON: %v", err)
		}
		if err := os.WriteFile(path, configJSON, 0644); err != nil {
			return fmt.Errorf("failed to write config to file %s: %v", path, err)
		}
	}
	if !cleanConfig {
		/* Decrypt passwords after writing */
		if err := process_decode(configValue); err != nil {
			return fmt.Errorf("failed to decode passwords in config entries: %v", err)
		}
	}
	return nil
}

func encrypt(text string) string {
	block, _ := aes.NewCipher(encryptionKey)
	gcm, _ := cipher.NewGCM(block)
	nonce := make([]byte, gcm.NonceSize())
	io.ReadFull(rand.Reader, nonce)
	ciphertext := gcm.Seal(nonce, nonce, []byte(text), nil)
	return base64.StdEncoding.EncodeToString(ciphertext)
}

func decrypt(text string) (string, error) {
	block, _ := aes.NewCipher(encryptionKey)
	gcm, _ := cipher.NewGCM(block)
	data, _ := base64.StdEncoding.DecodeString(text)
	nonceSize := gcm.NonceSize()
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	return string(plaintext), err
}
