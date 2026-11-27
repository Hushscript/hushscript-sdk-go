# Changelog

## Unreleased

### Features

* add the `vet_mode` (Veterinary mode, beta; en, en_au, en_uk, en_us, es, fr, de, de_ch; refused with `medical_mode` and in private mode) and `keep_disfluencies` (free, default off) transcription options, both echoed in the job options summary; the medical language list is synced.

### Bug Fixes

* translation targets no longer force speaker detection on: `Speakers` is sent as set; `TranslateTo` takes at most 15 distinct languages (an `invalid_option` error before any request); the API accepts every base language as a target (99 codes, no regional variants); `Medical` and `Vet` compare `Language` after trimming and lowercasing, as the API does; the generated client is synced with the API spec.

### Documentation

* README links point at the live pricing page and the SDK section of the developers page; the old `/pricing` and `/sdks` URLs redirect there.

## 1.0.0 (2026-10-13)


### Features

* initial release ([c5cf2eb](https://github.com/Hushscript/hushscript-sdk-go/commit/c5cf2ebed25213f4a69e81ee38a2305e8f9c6495))
