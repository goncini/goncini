# Symfony's constraints, as tags

Each of Symfony's validation constraints and what stands for it in a
goncini input: a `validate` tag, go-playground's or goncini's, or the
input's `Validate` method, for rules across fields or on the app's data.
Tags combine with commas: `validate:"required,min=3,max=120"`.

| Symfony | goncini |
|---|---|
| `NotBlank` | `required` |
| `NotNull` | a pointer, with `required` |
| `Blank`, `IsNull` | `isdefault` |
| `IsTrue`, `IsFalse` | `eq=true`, `eq=false` |
| `Type` | the field's Go type |
| `Length` | `min=n`, `max=n`, `len=n` on a string, in characters |
| `Count` | `min=n`, `max=n`, `len=n` on a slice or a map |
| `Email` | `email` |
| `Url` | `url`, `http_url`, `https_url` |
| `Regex` | `regex=pattern`, goncini's: RE2's syntax, a comma written `0x2C` and a bar `0x7C` |
| `Hostname` | `hostname`, `hostname_rfc1123`, `fqdn` |
| `Ip` | `ip`, `ipv4`, `ipv6` |
| `Cidr` | `cidr`, `cidrv4`, `cidrv6` |
| `Json` | `json` |
| `Uuid` | `uuid`, `uuid4`, …; or a `uid.UUID` field |
| `Ulid` | `ulid`; or a `uid.ULID` field |
| `CssColor` | `iscolor`, `hexcolor`, `rgb`, `rgba`, `hsl`, `hsla` |
| `MacAddress` | `mac` |
| `PasswordStrength` | `password_strength=score`, goncini's: 0 to 4, 2 without one, as Symfony estimates it |
| `NotCompromisedPassword` | `not_compromised=n`, goncini's: asks Have I Been Pwned, unless `validator.Config.SkipPwnedPasswords` |
| `UserPassword` | `Validate`, which compares it with the user's hash |
| `EqualTo`, `NotEqualTo` | `eq=v`, `ne=v`; `eqfield=F`, `nefield=F` for another field |
| `IdenticalTo`, `NotIdenticalTo` | `eq=v`, `ne=v` |
| `LessThan`, `LessThanOrEqual` | `lt=v`, `lte=v`; `ltfield=F`, `ltefield=F` |
| `GreaterThan`, `GreaterThanOrEqual` | `gt=v`, `gte=v`; `gtfield=F`, `gtefield=F` |
| `Range` | `min=a,max=b` on a number |
| `DivisibleBy` | `divisible_by=n`, goncini's |
| `Unique` | `unique` |
| `Positive`, `PositiveOrZero` | `gt=0`, `gte=0` |
| `Negative`, `NegativeOrZero` | `lt=0`, `lte=0` |
| `Date`, `DateTime`, `Time` | `datetime=2006-01-02`, `datetime=2006-01-02T15:04:05Z07:00`, `datetime=15:04:05`; or a `time.Time` field |
| `Timezone` | `timezone` |
| `Choice` | `oneof=a b c`; `oneofci` without case |
| `Country` | `iso3166_1_alpha2`, `iso3166_1_alpha3` |
| `Currency` | `iso4217` |
| `Language`, `Locale` | `bcp47_language_tag` |
| `Bic` | `bic` |
| `CardScheme` | `credit_card`, of any scheme |
| `Luhn` | `luhn_checksum` |
| `Iban` | `iban`, goncini's: its check digits, without spaces |
| `Isbn` | `isbn`, `isbn10`, `isbn13` |
| `Issn` | `issn` |
| `Isin` | `isin`, goncini's |
| `File`, `Image` | an `httpkernel.File`, checked in `Validate` |
| `Valid`, `All` | `dive`, which checks each item of a slice or a map |
| `Callback`, `Expression`, `When` | `Validate` |
| `AtLeastOneOf`, `Sequentially`, `Compound` | tags in a row, or `Validate` |
| `UniqueEntity` | `Validate`, which asks the app's store |

What isn't there, such as `Charset`, `WordCount`, `Week` and
`NoSuspiciousCharacters`, is a rule of the app's own: `Rules()` returns
go-playground's validator, whose `RegisterValidation` adds a tag.
