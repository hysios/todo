/*
Copyright © 2020 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"

	"github.com/hysios/todo/parser"
	"github.com/hysios/todo/printer"
	"github.com/spf13/cobra"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
)

var (
	cfgFile    string
	inputs     []string
	autoNumber bool
	rewrite    bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "todo",
	Short: "A brief description of your application",
	// Uncomment the following line if your bare application
	// has an action associated with it:
	Run: func(cmd *cobra.Command, args []string) {
		cwd, err := os.Getwd()
		if err != nil {
			log.Fatal(err)
		}

		if len(inputs) == 0 {
			inputs, err = lookupTodos(cwd)
			if err != nil {
				log.Fatal(err)
			}
		}

		for _, todoname := range inputs {
			todo, err := parseTodo(todoname)
			if err != nil {
				log.Fatal(err)
			}

			printTodo(todo)
			if rewrite {
				rewriteTodo(todoname, todo)
			}
		}

	},
}

func lookupTodos(dir string) ([]string, error) {
	lists, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var todos = make([]string, 0)

	for _, fi := range lists {
		if filepath.Ext(fi.Name()) == ".todo" {
			todos = append(todos, fi.Name())
		} else if filepath.Base(fi.Name()) == "TODO" {
			todos = append(todos, fi.Name())
		}
	}

	return todos, nil

}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)

		os.Exit(1)
	}
}

func parseTodo(filename string) (*parser.Todofile, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	todo, err := parser.Parse(filename, f)
	if err != nil {
		return nil, err
	}

	return todo, nil
}

var numRe = regexp.MustCompile(`^(\d+).`)

func printTodo(todo *parser.Todofile) {
	var (
		num   int = 1
		print     = printer.New(todo)
	)

	if autoNumber {
		print.AddPipe(regeneratorNumber(num))
	}
	print.Print()
}

func regeneratorNumber(num int) printer.PrinterFunc {
	return func(node *parser.Todoitem, w io.Writer) {
		if node.Type == parser.ItItem {
			ss := numRe.FindStringSubmatch(node.Text)
			if len(ss) == 0 { // not numbering
				prefix := strconv.Itoa(num) + ". "
				node.SetOffset(len(prefix))
				node.Text = prefix + node.Text
			} else {
				_num, err := strconv.Atoi(ss[1])
				if err != nil {
					log.Fatal(err)
				}

				oldns := strconv.Itoa(_num)
				ns := strconv.Itoa(num)
				if _num > num {
					num = _num
				}

				ofs := len(ns) - len(oldns)
				node.SetOffset(ofs)
				r := []rune(node.Text)
				if ofs > 0 {
					prefix := strconv.Itoa(num) + "."
					node.Text = prefix + string(r[len(ns):])
				} else if ofs < 0 {
					// prefix := strconv.Itoa(num)
					// node.Text = prefix + string(r[len(ns):])
				} else {
					prefix := strconv.Itoa(num)
					node.Text = prefix + string(r[len(ns):])
				}
			}
			num++
		}
	}
}

func rewriteTodo(filename string, todo *parser.Todofile) error {
	var (
		num   int = 1
		print     = printer.New(todo)
	)
	f, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	w := bufio.NewWriter(f)
	defer f.Close()
	if autoNumber {
		print.AddPipe(regeneratorNumber(num))
	}

	print.WriteTo(f)
	w.Flush()
	// f.Flush()
	return nil
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.todo.yaml)")
	rootCmd.PersistentFlags().StringSliceVarP(&inputs, "input", "i", nil, "todo file input list")
	rootCmd.PersistentFlags().BoolVarP(&autoNumber, "auto-number", "n", false, "auto numbering todo items")
	rootCmd.PersistentFlags().BoolVarP(&rewrite, "rewrite", "w", false, "rewrite todolist to file")

}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".todo" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".todo")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
