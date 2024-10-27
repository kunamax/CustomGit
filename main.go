package main

import (
	"bufio"
	"crypto/sha256"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

func prependToFile(filePath, header string) error {
	originalFile, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("could not open file: %w", err)
	}
	defer originalFile.Close()

	tempFile, err := os.CreateTemp("", "tempfile-*")
	if err != nil {
		return fmt.Errorf("could not create temporary file: %w", err)
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	if _, err := tempFile.WriteString(header + "\n"); err != nil {
		return fmt.Errorf("could not write header: %w", err)
	}

	if _, err := io.Copy(tempFile, originalFile); err != nil {
		return fmt.Errorf("could not copy original content: %w", err)
	}

	tempFile.Close()
	originalFile.Close()

	if err := os.Rename(tempFile.Name(), filePath); err != nil {
		return fmt.Errorf("could not rename temp file to original: %w", err)
	}

	return nil
}

func printHelp(d map[string]string, names []string) {
	fmt.Println("These are SVCS commands:")
	for _, key := range names {
		fmt.Print(key, d[key])
		fmt.Println()
	}
}

func getFunctionSpec(d map[string]string, key string) {
	if val, ok := d[key]; ok {
		fmt.Println(strings.Trim(val, " "))
	} else {
		fmt.Printf("'%v' is not a SVCS command.", key)
	}
}

func handleConfig(value string) string {
	if value == "None" {
		fmt.Println("Please, tell me who you are.")
		fmt.Scanln(&value)
	} else {
		fmt.Printf("The username is %v.", value)
	}
	return value
}

func config() {
	var name string
	if len(os.Args) == 3 {
		name = handleConfig(os.Args[2])
	} else {
		dat, err := os.ReadFile("./vcs/config.txt")
		if err != nil || string(dat) == "" {
			name = handleConfig("None")
		} else {
			fmt.Printf("The username is %v.", strings.Trim(string(dat), "\n"))
		}
	}
	if err := os.WriteFile("./vcs/config.txt", []byte(name), 0644); err != nil {
		panic(err)
	}
}

func logs() {
	if dat, err := os.ReadFile("./vcs/log.txt"); err != nil || string(dat) == "" {
		fmt.Println("No commits yet.")
	} else {
		fmt.Println(string(dat))
	}

	f, err := os.OpenFile("./vcs/log.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	defer f.Close()
}

func add() {
	if len(os.Args) == 3 {
		if _, err := os.Open(os.Args[2]); err != nil {
			fmt.Printf("Can't find '%v'.", os.Args[2])
			os.Exit(3)
		}

		f, err := os.OpenFile("./vcs/index.txt", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
		if err != nil {
			panic(err)
		}
		defer f.Close()
		if _, err = f.WriteString(os.Args[2] + "\n"); err != nil {
			panic(err)
		}
		fmt.Printf("The file '%v' is tracked.", os.Args[2])
	} else {
		dat, err := os.ReadFile("./vcs/index.txt")
		if err != nil || string(dat) == "" {
			fmt.Println("Add a file to the index.")
			var fileName string
			fmt.Scanln(&fileName)
			f, err := os.OpenFile("./vcs/index.txt", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
			if err != nil {
				panic(err)
			}
			defer f.Close()

			if _, err := os.Open(fileName); err != nil {
				fmt.Printf("Can't find '%v'.", fileName)
				os.Exit(3)
			}

			if _, err = f.WriteString(fileName + "\n"); err != nil {
				panic(err)
			}
			fmt.Println("")
		} else {
			fmt.Println("Tracked files:")
			//scanner := bufio.NewScanner(file)
			//for scanner.Scan() {
			//	fmt.Println(scanner.Text())
			//}
			fmt.Println(string(dat))
		}
	}
}

func commit() {
	if len(os.Args) == 2 {
		fmt.Println("Message was not passed.")
		return
	}
	message := strings.ReplaceAll(os.Args[2], "\"", "")
	os.Mkdir("./vcs/commits", 0777)

	commitHash := sha256.New()
	commitHash.Write([]byte(message))

	commitHashString := fmt.Sprintf("%x", commitHash.Sum(nil))

	filesIndex := map[string]bool{}

	file, err := os.Open("./vcs/index.txt")
	if err != nil {
		panic(err)
	}
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		filesIndex[scanner.Text()] = true
	}

	filesHash := map[string]string{}
	for k, _ := range filesIndex {
		dat, err := os.ReadFile(k)
		if err != nil {
			panic(err)
		}
		fileHash := sha256.New()
		fileHash.Write(dat)

		filesHash[k] = string(fileHash.Sum(nil))
	}

	var latestCommit string
	var counter int = 0

	entries, err := os.ReadDir("./vcs/commits")
	if err != nil {
		log.Fatal(err)
	}

	for _, entry := range entries {
		if entry.Name() == ".DS_Store" {
			counter--
		}
		if entry.IsDir() && counter == 0 {
			latestCommit = entry.Name()
		}
		counter++
	}

	var difference bool = false
	if latestCommit != "" {
		for k, val := range filesHash {
			data, err := os.ReadFile("./vcs/commits/" + latestCommit + "/" + k)
			if err != nil {
				panic(err)
			}
			fileHash := sha256.New()
			fileHash.Write(data)

			if val != string(fileHash.Sum(nil)) {
				difference = true
			}
		}
	} else {
		difference = true
	}

	if difference {
		erro := os.Mkdir("./vcs/commits/"+commitHashString, 0777)
		if erro != nil {
		}

		for f, _ := range filesIndex {
			data, err := os.ReadFile(f)
			if err != nil {
				panic(err)
			}
			erro := os.WriteFile("./vcs/commits/"+commitHashString+"/"+f, data, 0644)
			if erro != nil {
				panic(erro)
			}
		}
		addLog(commitHashString, message)
		fmt.Println("Changes are committed.")
	} else {
		fmt.Println("Nothing to commit.")
	}

}

func addLog(commitHashString string, message string) {
	data, err := os.ReadFile("./vcs/config.txt")
	if err != nil {
		panic(err)
	}
	name := string(data)

	f, err := os.OpenFile("./vcs/log.txt", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	header := "commit " + commitHashString + "\nAuthor: " + name + "\n" + message + "\n"

	err = prependToFile("./vcs/log.txt", header)
	if err != nil {
		fmt.Println("Error:", err)
	}
}

func checkout() {
	if len(os.Args) == 2 {
		fmt.Println("Commit id was not passed.")
		return
	}
	commitsAvailable := map[string]bool{}

	entries, err := os.ReadDir("./vcs/commits")
	if err != nil {
		log.Fatal(err)
	}

	for _, entry := range entries {
		if entry.Name() != ".DS_Store" {
			commitsAvailable[entry.Name()] = true
		}
	}

	isIn, _ := commitsAvailable[os.Args[2]]
	if !isIn {
		fmt.Println("Commit does not exist.")
		return
	} else {
		filesIndex := map[string]bool{}

		file, err := os.Open("./vcs/index.txt")
		if err != nil {
			panic(err)
		}
		scanner := bufio.NewScanner(file)

		for scanner.Scan() {
			filesIndex[scanner.Text()] = true
		}

		for f, _ := range filesIndex {
			data, err := os.ReadFile("./vcs/commits/" + os.Args[2] + "/" + f)
			if err != nil {
				panic(err)
			}
			erro := os.WriteFile(f, data, 0644)
			if erro != nil {
				panic(erro)
			}
		}
		fmt.Println("Switched to commit " + os.Args[2] + ".")
	}
}

func main() {
	//home, _ := os.UserHomeDir()
	//home = filepath.Join(home, "Documents/OOP/Version Control System (Go)/Version Control System (Go)/task")
	//err := os.Chdir(home)
	//if err != nil {
	//	panic(err)
	//}

	err := os.MkdirAll("./vcs", os.ModePerm)
	if err != nil {
		panic(err)
	}

	if len(os.Args) != 2 {
		errors.New("expected 1 argument")
	}

	d := map[string]string{
		"config":   "     Get and set a username.",
		"add":      "        Add a file to the index.",
		"log":      "        Show commit logs.",
		"commit":   "     Save changes.",
		"checkout": "   Restore a file.",
	}

	names := []string{"config", "add", "log", "commit", "checkout"}

	var help bool
	flag.BoolVar(&help, "help", true, "Help")

	flag.Parse()

	if len(os.Args) == 1 {
		printHelp(d, names)
	} else {
		command := os.Args[1]

		switch command {
		case "--help":
			printHelp(d, names)
		case "config":
			config()
		case "add":
			add()
		case "log":
			logs()
		case "commit":
			commit()
		case "checkout":
			checkout()
		default:
			fmt.Printf("'%v' is not a SVCS command.", command)
		}
	}
}
