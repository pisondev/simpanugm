# Inter Latin variable subset

`inter-latin-variable.woff2` is the existing build's Inter Latin subset (48,432 bytes),
previously self-hosted at runtime by `next/font/google`. It is now a source asset loaded
by `next/font/local`, removing network access at build time as well.

Upstream: https://github.com/rsms/inter

License: SIL Open Font License 1.1, distributed in `OFL.txt` from
https://raw.githubusercontent.com/google/fonts/main/ofl/inter/OFL.txt

Weight range: 100 to 900, normal style. System UI supplies glyphs outside this Latin
subset. No font transformation or new font dependency is introduced.
