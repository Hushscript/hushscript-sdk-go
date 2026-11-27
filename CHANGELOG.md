# Changelog

## [1.2.0](https://github.com/Hushscript/hushscript-sdk-go/compare/v1.2.1...v1.2.0) (2026-11-27)


### Features

* initial release ([c5cf2eb](https://github.com/Hushscript/hushscript-sdk-go/commit/c5cf2ebed25213f4a69e81ee38a2305e8f9c6495))
* sync with the api spec ([ab936be](https://github.com/Hushscript/hushscript-sdk-go/commit/ab936be714650a9ae5243610f5dff4c12902029b))


### Bug Fixes

* avoid live context reflection in redaction test ([5495ecb](https://github.com/Hushscript/hushscript-sdk-go/commit/5495ecb4ca5d749634f80651c9c5f4664b30d389))
* **deps:** update dependencies to latest ([9a67b7b](https://github.com/Hushscript/hushscript-sdk-go/commit/9a67b7bfc5012f483dd442418743b18156793b7f))
* name inline object schemas in the generated client ([af0c28e](https://github.com/Hushscript/hushscript-sdk-go/commit/af0c28ef40e72557b4a6597a5d9b0cc41ee22121))
* no duplicate part-urls fetch when another part's fetch just finished ([924ca1c](https://github.com/Hushscript/hushscript-sdk-go/commit/924ca1c3cf556a02ef7c0ad6c47efa5f2696144d))
* part-urls refetch dedup by page ([29ad66c](https://github.com/Hushscript/hushscript-sdk-go/commit/29ad66c4bef8b7ee61ce8ac7b3d4f2fcbcf3dfe6))
* refresh and stale-page refetch skip by page, not url ([d23a348](https://github.com/Hushscript/hushscript-sdk-go/commit/d23a3482e97879e43c655249c0f9dd05170cffaf))
* sync the api spec with production and pin conformance v1.1.0 ([0285d46](https://github.com/Hushscript/hushscript-sdk-go/commit/0285d4668df0cc9569ca7daec61a048d1c9aeba1))
* sync the spec with the backend: org summary drops is_owner and owner_user_id (never sent) and adds matter_code_policy, rebill_rate_cents_per_hour and letterhead; upload mode is standard or private; six more upload options; rotate always returns expires_at ([12e4293](https://github.com/Hushscript/hushscript-sdk-go/commit/12e42930000a14e13d2b2dee4038a49867e8090c))
* translation no longer forces speakers, cap targets at 15, sync spec ([78f5259](https://github.com/Hushscript/hushscript-sdk-go/commit/78f5259b835cf2ef22575987b5321b63c7879901))

## [1.2.1](https://github.com/Hushscript/hushscript-sdk-go/compare/v1.2.0...v1.2.1) (2026-11-27)


### Bug Fixes

* sync the spec with the backend: org summary drops is_owner and owner_user_id (never sent) and adds matter_code_policy, rebill_rate_cents_per_hour and letterhead; upload mode is standard or private; six more upload options; rotate always returns expires_at ([12e4293](https://github.com/Hushscript/hushscript-sdk-go/commit/12e42930000a14e13d2b2dee4038a49867e8090c))

## [1.2.0](https://github.com/Hushscript/hushscript-sdk-go/compare/v1.1.1...v1.2.0) (2026-11-27)

### Features

* add the `vet_mode` (Veterinary mode, beta; en, en_au, en_uk, en_us, es, fr, de, de_ch; refused with `medical_mode` and in private mode) and `keep_disfluencies` (free, default off) transcription options, both echoed in the job options summary; the medical language list is synced.

### Bug Fixes

* translation targets no longer force speaker detection on: `Speakers` is sent as set; `TranslateTo` takes at most 15 distinct languages (an `invalid_option` error before any request); the API accepts every base language as a target (99 codes, no regional variants); `Medical` and `Vet` compare `Language` after trimming and lowercasing, as the API does; the generated client is synced with the API spec.
* the API spec is synced with production: `translation_targets` no longer declares `maxItems` (the API counts the 15-target cap after dedup, as the SDK already does); the conformance suite is pinned to the `v1.1.0` tag.

### Documentation

* README links point at the live pricing page and the SDK section of the developers page; the old `/pricing` and `/sdks` URLs redirect there.

## [1.1.1](https://github.com/Hushscript/hushscript-sdk-go/compare/v1.1.0...v1.1.1) (2026-10-31)


### Bug Fixes

* **deps:** update dependencies to latest ([9a67b7b](https://github.com/Hushscript/hushscript-sdk-go/commit/9a67b7bfc5012f483dd442418743b18156793b7f))

## [1.1.0](https://github.com/Hushscript/hushscript-sdk-go/compare/v1.0.1...v1.1.0) (2026-10-23)


### Features

* sync with the api spec ([ab936be](https://github.com/Hushscript/hushscript-sdk-go/commit/ab936be714650a9ae5243610f5dff4c12902029b))


### Bug Fixes

* avoid live context reflection in redaction test ([5495ecb](https://github.com/Hushscript/hushscript-sdk-go/commit/5495ecb4ca5d749634f80651c9c5f4664b30d389))
* name inline object schemas in the generated client ([af0c28e](https://github.com/Hushscript/hushscript-sdk-go/commit/af0c28ef40e72557b4a6597a5d9b0cc41ee22121))

## [1.0.1](https://github.com/Hushscript/hushscript-sdk-go/compare/v1.0.0...v1.0.1) (2026-10-14)


### Bug Fixes

* no duplicate part-urls fetch when another part's fetch just finished ([924ca1c](https://github.com/Hushscript/hushscript-sdk-go/commit/924ca1c3cf556a02ef7c0ad6c47efa5f2696144d))
* refresh and stale-page refetch skip by page, not url ([d23a348](https://github.com/Hushscript/hushscript-sdk-go/commit/d23a3482e97879e43c655249c0f9dd05170cffaf))

## 1.0.0 (2026-10-13)


### Features

* initial release ([c5cf2eb](https://github.com/Hushscript/hushscript-sdk-go/commit/c5cf2ebed25213f4a69e81ee38a2305e8f9c6495))
