# SteamAuth

Blatently ported, warts and all from https://github.com/geel9/SteamAuth

A Go library for logging into Steam with SteamGuard support.

## Functionality

Currently, like the original, this library can:

 * Generate login codes for a given shared secret
 * Login to a user account
 * Link and activate a new mobile authenticator to a user account after logging in
 * Remove itself from an account
 * Fetch, accept, and deny mobile confirmations

## Usage Notes

If you already have a `SharedSecret` just instantiate a `SteamGuardAccount` and call GenerateSteamGuardCode()

To remove the authenticator from your account you will need a working session (so log in with `UserLogin` or load a saved SteamGuardAccount json blob)

## Usage

### Authenticating

		auth := steamauth.NewUserLogin("username", "password")
		res, err := auth.DoLogin()
		// err usually means something went wrong in the library or connecting to steam
		switch res {
		case steamauth.NeedCaptcha:
			fmt.Printf("Requires captcha: %s\n", userLogin.CaptchaURL())
		case steamauth.Need2FA:
			fmt.Println("Need two factor code, get this from SteamGuard on your phone (or SteamGuardAccount if it's registered)")
		case steamauth.NeedEmail:
			fmt.Println("Code was sent to your email")
		case steamauth.LoginOkay:
			fmt.Println("Logged in!")
		}

### Begin registration

		linker := steamauth.NewAuthenticatorLinker(userLogin.Session)
		linkres, err := linker.AddAuthenticator()
		// Again err usually means something went wrong in the library or connecting to steam
		switch linkres {
		case steamauth.MustProvidePhoneNumber:
			fmt.Println("Account doesn't have a phone number associated with it and you didn't provide one")
		case steamauth.MustRemovePhoneNumber:
			fmt.Println("Account already has a phone number associated with it and you provided one")
		case steamauth.AwaitingFinalization:
			fmt.Println("A message has been sent to the given mobile number, call FinalizeAddAuthenticator(smscode)")
		}

### Finalize registration

		finres, err := linker.FinalizeAddAuthenticator(code)
		switch finres {
		case steamauth.BadSMSCode:
			fmt.Println("You done typoed son")
		case steamauth.UnableToGenerateCorrectCodes:
			fmt.Println("Steam doesn't like us, even after 30 tries")
		case steamauth.Success
			fmt.Println("Everything is awsome")
		}

### Save state

Once you've finalized your registration you should absolutly save a copy of the `SteamGuardAccount` instance

		fmt.Println(linker.LinkedAccount.Export())

## Example

Look in `examples` for an example that should authenticate and register itself with a given account

