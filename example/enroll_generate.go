package main

import (
	"bufio"
	"fmt"
	"os"
	"github.com/freman/go-steamauth"
	"strings"
)

func prompt(msg string) string {
	text := ""
	reader := bufio.NewReader(os.Stdin)
	for text == "" {
		fmt.Printf("%s: ", msg)
		text, _ = reader.ReadString('\n')
		text = strings.TrimSpace(text)
	}
	return text
}

func getCode() string {
	return strings.ToUpper(prompt("Code: "))
}

type logger struct{}

func (l logger) Output(calldepth int, s string) error {
	fmt.Printf("\t[%d]: %s\n", calldepth, s)
	return nil
}

func main() {
	fmt.Println("Steamauth Demo")
	fmt.Println("--------------")

	account := steamauth.SteamGuardAccount{}
	if file, err := os.Open("steam_data.json"); err == nil {
		defer file.Close()
		if err = account.Load(file); err != nil {
			fmt.Println("Problem parsing steam_data.json,", err)
		}
	} else {
		fmt.Println(err)
	}

	if account.FullyEnrolled {
		fmt.Println("Already enrolled")
	} else {
		username := prompt("Steam username: ")
		password := prompt("Steam password: ")

		logger := logger{}
		steamauth.SetLogger(logger)
		userLogin := steamauth.NewUserLogin(username, password)

		done := false
		for !done {
			res, err := userLogin.DoLogin()

			if err != nil {
				panic(err)
			}

			switch res {
			case steamauth.NeedCaptcha:
				fmt.Printf("Requires captcha, go to %s to get it\n", userLogin.CaptchaURL())
				userLogin.CaptchaText = getCode()
			case steamauth.Need2FA:
				fmt.Println("Need two factor code")
				userLogin.TwoFactorCode = getCode()
			case steamauth.NeedEmail:
				fmt.Println("Please check your email for the code")
				userLogin.EmailCode = getCode()
			case steamauth.LoginOkay:
				done = true
				fmt.Println("Logged in!")
			default:
				panic(res)
			}
		}

		linker := steamauth.NewAuthenticatorLinker(userLogin.Session)
		linkres, err := linker.AddAuthenticator()
		if err != nil {
			panic(err)
		}

		if linkres == steamauth.MustProvidePhoneNumber {
			fmt.Println("Your account needs to be linked to a mobile phone number")
			number := prompt("International format phone number starting with +")
			linker.PhoneNumber = number
			linkres, err = linker.AddAuthenticator()
			if err != nil {
				panic(err)
			}
		}

		if linkres == steamauth.AwaitingFinalization {
			fmt.Println("Enter code")
			code := getCode()
			finres, err := linker.FinalizeAddAuthenticator(code)
			if err != nil {
				panic(err)
			}
			fmt.Println(finres)

			failed := false
			file, err := os.Create("steam_data.json")
			defer file.Close()
			if err != nil {
				failed = true
				fmt.Println(err)
			} else if err = account.Save(file); err != nil {
				fmt.Println("Problem saving steam_data.json,", err)
				failed = true
			}

			if failed {
				fmt.Println("Cannot save to steam_data.json, please save this json response")
				fmt.Println(linker.LinkedAccount.Export())
				return
			}
			account = linker.LinkedAccount
		} else {
			panic(err)
		}
	}

	fmt.Println("Your steamguard code:", account.GenerateSteamGuardCode())

	fmt.Printf("%#v\n", account.FetchConfirmations)
}
