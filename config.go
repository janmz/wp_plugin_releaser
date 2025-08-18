package main

/*
 * Dieses Modul enthält eine Funktion zum Management von Config-Dateien mit sicheren Passwörtern.
 *
 * Version 1.0
 *
 * Autor: Jan Neuhaus, VAYA Consulting, https://vaya-consultig.de/development/ https://github.com/janmz
 *
 * Funktionen:
 * - loadConfig(): Lädt die Konfiguration aus einer Datei und verarbeitet sie.
 *
 * Abhängigkeiten:
 * - hardware-id.go: Damit Passwörter nicht auf anderen Systemen entschlüsselt werden können, wird mit dieser Datei ein systemabhängiger Schlüssel erzeugt.
 * - i18n.go: Für die Internationalisierung der Fehlermeldungen
 */

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
var PASSWORD_IS_SECURE string    // String to be written
var PASSWORD_IS_SECURE_en string // String to be recognized
var PASSWORD_IS_SECURE_de string // String to be recognized

var encryptionKey []byte
var initialized = false

/*
 * Reading a JSON file containing data for the config struct, where
 * passwords are encrypted and decrypted, and if the cleanConfig parameter
 * is set, a file with unencrypted passwords is written.
 * @param config	Ist eine Struktur, die die einzulesende Config-Datei aufnehmen wird
 * @param version	Wenn in der Struktur eine Variable Version vorhanden ist, wird diese aktuell gehalten
 * @param path		Pfad unter dem die config-Datei gespeichert ist
 * @param cleanConfig	Für den Fall, dass man die Passwörter doch nochmal im Klartext braucht, kann damit erzwungen werden, die Datei mit Klartextpasswörtern zu schreiben.
 * @return error	Fehlermeldung, wenn die Config-Datei nicht gelesen werden konnte
 */
func loadConfig(config interface{}, version int, path string, cleanConfig bool, getHardwareID_func ...func() (uint64, error)) error {

	var file []byte

	if len(getHardwareID_func) > 0 {
		config_init(getHardwareID_func[0])
	} else {
		config_init(getHardwareID)
	}

	_, err := os.Stat(path)
	if !os.IsNotExist(err) {
		file, err = os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read config file: %v", err)
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

	if err := updateDefaultValues(configValue); err != nil {
		return fmt.Errorf("failed to set default config entries: %v", err)
	}

	if err := json.Unmarshal(file, config); err != nil {
		return fmt.Errorf("failed to parse config file: %v", err)
	}
	changed := false
	if err := updateVersionAndPasswords(configValue, version, &changed); err != nil {
		return fmt.Errorf("failed to check config entries: %v", err)
	}
	if cleanConfig {
		/* Decrypt passwords before writing */
		if err := decodePasswords(configValue); err != nil {
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
		if err := decodePasswords(configValue); err != nil {
			return fmt.Errorf("failed to decode passwords in config entries: %v", err)
		}
	}
	return nil
}

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
func config_init(getHardwareID_func func() (uint64, error)) {
	if !initialized {
		// Generate encryption key based on Hardware IS
		hardwareID, err := getHardwareID_func()
		if err != nil {
			log.Fatalf(t("config.hardware_id_failed"))
		}
		randGenSeeded := mathRand.NewSource(int64(hardwareID))
		encryptionKey = make([]byte, 32)
		for i := range encryptionKey {
			encryptionKey[i] = byte(randGenSeeded.Int63() >> 16 & 0xff)
		}
		curr_lang := getCurrentLanguage()
		setLanguage("de")
		PASSWORD_IS_SECURE_de = t("app.password_message")
		setLanguage("en")
		PASSWORD_IS_SECURE_en = t("app.password_message")
		setLanguage(curr_lang)
		PASSWORD_IS_SECURE = t("app.password_message")
	}
	initialized = true
}

/*
 * Go through the structure and set the default values present
 * in the annotations
 */
func updateDefaultValues(v reflect.Value) error {
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return nil
	}
	type_info := v.Type()
	// Iterate through all fields
	for i := 0; i < type_info.NumField(); i++ {
		field := type_info.Field(i)
		fieldValue := v.Field(i)
		if field.Type.Kind() == reflect.Struct {
			if err := updateDefaultValues(fieldValue); err != nil {
				return fmt.Errorf(t("config.default_error"), err)
			}
		} else if field.Type.Kind() == reflect.Slice {
			for i := 0; i < fieldValue.Len(); i++ {
				if fieldValue.Index(i).Kind() == reflect.Struct {
					if err := updateDefaultValues(fieldValue.Index(i)); err != nil {
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
						return fmt.Errorf(t("config.default_error"), err)
					}
					fieldValue.SetInt(int64(value))
				case reflect.Bool:
					boolValue, err := strconv.ParseBool(defaultValue)
					if err != nil {
						return fmt.Errorf(t("config.default_error"), err)
					}
					fieldValue.SetBool(boolValue)
				default:
					return fmt.Errorf(t("config.default_unsupported"), fieldValue.Kind())
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
func updateVersionAndPasswords(v reflect.Value, version int, changed *bool) error {
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
			if err := updateVersionAndPasswords(fieldValue, version, changed); err != nil {
				return err
			}
		} else if field.Type.Kind() == reflect.Slice {
			//fmt.Printf("Slice[0..%d]\n", fieldValue.Len()-1)
			for i := 0; i < fieldValue.Len(); i++ {
				//fmt.Printf("Slice-Element %d:\n", i)
				if fieldValue.Index(i).Kind() == reflect.Struct {
					if err := updateVersionAndPasswords(fieldValue.Index(i), version, changed); err != nil {
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
					*changed = true
				}
			}
			// Password handling
			if strings.HasSuffix(field.Name, "SecurePassword") {
				pw_prefix := strings.TrimSuffix(field.Name, "SecurePassword")
				for j := 0; j < t.NumField(); j++ {
					if t.Field(j).Name == pw_prefix+"Password" {
						field2Value := v.Field(j)
						if field2Value.String() != PASSWORD_IS_SECURE_de && field2Value.String() != PASSWORD_IS_SECURE_en {
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
func decodePasswords(v reflect.Value) error {
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return nil
	}
	type_info := v.Type()
	// Iterate through all fields
	for i := 0; i < type_info.NumField(); i++ {
		field := type_info.Field(i)
		fieldValue := v.Field(i)

		// Process recursively nested structures
		if field.Type.Kind() == reflect.Struct {
			if err := decodePasswords(fieldValue); err != nil {
				return err
			}
		} else if field.Type.Kind() == reflect.Slice {
			for i := 0; i < fieldValue.Len(); i++ {
				if fieldValue.Index(i).Kind() == reflect.Struct {
					if err := decodePasswords(fieldValue.Index(i)); err != nil {
						return err
					}
				}
			}
		} else {
			// Password processing
			if strings.HasSuffix(field.Name, "SecurePassword") {
				pw_prefix := strings.TrimSuffix(field.Name, "SecurePassword")
				for j := 0; j < type_info.NumField(); j++ {
					if type_info.Field(j).Name == pw_prefix+"Password" {
						field2Value := v.Field(j)
						password, err := decrypt(fieldValue.String())
						if err != nil {
							return fmt.Errorf(t("config.decrypt_failed", pw_prefix), err)
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
