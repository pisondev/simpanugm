# Demo photo model checks

## Baseline categories as shipped

Scores are the live ML endpoint response for the category prompts listed in `data/seed/categories.yaml`.

| Image | Type | Expected | Top 1 | Top 1 score | Top 2 | Top 2 score | Top 3 | Top 3 score |
|---|---|---|---|---:|---|---:|---|---:|
| hero-earbuds-found.jpg | hero_found | earphone | charger | 0.7866 | power-bank | 0.1070 | earphone | 0.0180 |
| hero-wallet-found.jpg | hero_found | dompet | tempat-kartu | 0.9507 | dompet | 0.0397 | pouch | 0.0065 |
| hero-keys-found.jpg | hero_found | kunci-motor | ponsel | 0.1215 | kartu-atm | 0.1024 | sim | 0.0972 |
| hero-calculator-found.jpg | hero_found | kalkulator | kalkulator | 0.9843 | tablet | 0.0066 | laptop | 0.0025 |
| hero-umbrella-found.jpg | hero_found | payung | pakaian-lain | 0.4645 | lainnya | 0.1695 | ransel | 0.0883 |
| hero-tumbler-found.jpg | hero_found | tumbler | ktm | 0.1170 | dokumen | 0.0969 | ponsel | 0.0919 |
| hero-jacket-found.jpg | hero_found | jaket | ponsel | 0.1298 | kartu-atm | 0.1107 | ktm | 0.0869 |
| dup-glasses-1.jpg | hero_found | kacamata | kacamata | 0.9990 | lainnya | 0.0002 | jas-almamater | 0.0002 |
| dup-glasses-2.jpg | hero_found | kacamata | ponsel | 0.1285 | dokumen | 0.0966 | ktm | 0.0882 |
| hero-flashdisk-found.jpg | hero_found | flashdisk | flashdisk | 0.9577 | power-bank | 0.0280 | charger | 0.0115 |
| hero-powerbank-found.jpg | hero_found | power-bank | laptop | 0.5501 | power-bank | 0.3134 | tablet | 0.0560 |
| hero-book-found.jpg | hero_found | buku | ponsel | 0.1496 | sim | 0.1033 | dokumen | 0.0933 |
| hero-backpack-found.jpg | hero_found | ransel | tas-laptop | 0.8815 | ransel | 0.0810 | laptop | 0.0158 |
| hero-helmet-found.jpg | hero_found | helm | helm | 0.5500 | earphone | 0.2935 | perhiasan | 0.0326 |
| bg-earphone-01.jpg | background | earphone | tablet | 0.4638 | earphone | 0.1652 | ponsel | 0.1308 |
| bg-earphone-02.jpg | background | earphone | earphone | 0.6996 | charger | 0.2355 | perhiasan | 0.0151 |
| bg-wallet-01.jpg | background | dompet | ponsel | 0.1045 | dokumen | 0.0974 | kartu-atm | 0.0905 |
| bg-wallet-02.jpg | background | dompet | pouch | 0.8647 | tote-bag | 0.0297 | dompet | 0.0192 |
| bg-keys-01.jpg | background | kunci-motor | flashdisk | 0.2062 | kunci-lain | 0.1961 | kunci-motor | 0.1829 |
| bg-keys-02.jpg | background | kunci-motor | kunci-lain | 0.5482 | kunci-motor | 0.3032 | kartu-atm | 0.0540 |
| bg-calc-01.jpg | background | kalkulator | kalkulator | 0.9984 | ponsel | 0.0003 | laptop | 0.0002 |
| bg-umbrella-01.jpg | background | payung | ponsel | 0.1292 | sim | 0.1184 | ktm | 0.0836 |
| bg-tumbler-01.jpg | background | tumbler | kartu-lain | 0.1433 | botol-minum | 0.1159 | dokumen | 0.0981 |
| bg-tumbler-02.jpg | background | tumbler | lainnya | 0.3044 | jas-almamater | 0.2986 | ponsel | 0.0971 |
| bg-jacket-01.jpg | background | jaket | dokumen | 0.1107 | kartu-atm | 0.0930 | ponsel | 0.0919 |
| bg-glasses-01.jpg | background | kacamata | kacamata | 0.9974 | lainnya | 0.0011 | perhiasan | 0.0005 |
| bg-flashdisk-01.jpg | background | flashdisk | flashdisk | 0.9577 | power-bank | 0.0280 | charger | 0.0115 |
| bg-powerbank-01.jpg | background | power-bank | laptop | 0.6877 | tablet | 0.1795 | flashdisk | 0.0644 |
| bg-book-01.jpg | background | buku | buku | 0.4500 | perhiasan | 0.2284 | alat-tulis | 0.1735 |
| bg-book-02.jpg | background | buku | buku | 0.4082 | alat-tulis | 0.2762 | lainnya | 0.0792 |
| bg-bag-01.jpg | background | ransel | sim | 0.1206 | ponsel | 0.1086 | lainnya | 0.0745 |
| bg-bag-02.jpg | background | ransel | tote-bag | 0.9924 | tas-laptop | 0.0067 | ransel | 0.0007 |
| bg-helmet-01.jpg | background | helm | sim | 0.1149 | ktm | 0.1081 | dokumen | 0.1043 |
| bg-watch-01.jpg | background | jam-tangan | jam-tangan | 0.9982 | perhiasan | 0.0013 | lainnya | 0.0002 |
| bg-watch-02.jpg | background | jam-tangan | flashdisk | 0.4702 | jam-tangan | 0.2903 | power-bank | 0.1180 |
| bg-mouse-01.jpg | background | mouse | laptop | 0.1994 | tablet | 0.1701 | jam-tangan | 0.1147 |
| bg-cable-01.jpg | background | kabel-charger | power-bank | 0.3540 | charger | 0.1770 | perhiasan | 0.0965 |
| bg-pouch-01.jpg | background | kotak-pensil | sim | 0.1369 | lainnya | 0.1030 | ponsel | 0.0900 |
| bg-shoes-01.jpg | background | sepatu | lainnya | 0.2668 | pakaian-lain | 0.2657 | earphone | 0.2556 |
| bg-hat-01.jpg | background | topi | ponsel | 0.1169 | flashdisk | 0.0975 | kartu-atm | 0.0889 |
| hero-earbuds-lost.jpg | hero_lost | earphone | charger | 0.7730 | power-bank | 0.1741 | sim | 0.0122 |
| hero-wallet-lost.jpg | hero_lost | dompet | tempat-kartu | 0.5676 | dompet | 0.4118 | kartu-atm | 0.0069 |
| hero-keys-lost.jpg | hero_lost | kunci-motor | lainnya | 0.1207 | dokumen | 0.1125 | buku | 0.0721 |
| hero-calculator-lost.jpg | hero_lost | kalkulator | kalkulator | 0.9814 | tablet | 0.0055 | laptop | 0.0042 |
| hero-umbrella-lost.jpg | hero_lost | payung | pakaian-lain | 0.5883 | lainnya | 0.1433 | pouch | 0.0798 |
| hero-tumbler-lost.jpg | hero_lost | tumbler | dokumen | 0.1385 | lainnya | 0.1234 | ponsel | 0.0747 |
| hero-jacket-lost.jpg | hero_lost | jaket | dokumen | 0.1496 | lainnya | 0.0986 | laptop | 0.0662 |
| hero-flashdisk-lost.jpg | hero_lost | flashdisk | flashdisk | 0.9726 | power-bank | 0.0216 | charger | 0.0045 |
| hero-powerbank-lost.jpg | hero_lost | power-bank | power-bank | 0.3263 | tablet | 0.2592 | laptop | 0.0836 |
| hero-book-lost.jpg | hero_lost | buku | dokumen | 0.1122 | lainnya | 0.1055 | buku | 0.0769 |
| hero-backpack-lost.jpg | hero_lost | ransel | tas-laptop | 0.8499 | ransel | 0.0974 | laptop | 0.0125 |
| hero-helmet-lost.jpg | hero_lost | helm | helm | 0.9640 | earphone | 0.0238 | lainnya | 0.0023 |

## Prompt revision 1

Scores are the live ML endpoint response for the category prompts listed in `data/seed/categories.yaml`.

| Image | Type | Expected | Top 1 | Top 1 score | Top 2 | Top 2 score | Top 3 | Top 3 score |
|---|---|---|---|---:|---|---:|---|---:|
| hero-earbuds-found.jpg | hero_found | earphone | charger | 0.7866 | power-bank | 0.1070 | earphone | 0.0180 |
| hero-wallet-found.jpg | hero_found | dompet | tempat-kartu | 0.9507 | dompet | 0.0397 | pouch | 0.0065 |
| hero-keys-found.jpg | hero_found | kunci-motor | ponsel | 0.1215 | kartu-atm | 0.1024 | sim | 0.0972 |
| hero-calculator-found.jpg | hero_found | kalkulator | kalkulator | 0.9843 | tablet | 0.0066 | laptop | 0.0025 |
| hero-umbrella-found.jpg | hero_found | payung | pakaian-lain | 0.4645 | lainnya | 0.1695 | ransel | 0.0883 |
| hero-tumbler-found.jpg | hero_found | tumbler | ktm | 0.1170 | dokumen | 0.0969 | ponsel | 0.0919 |
| hero-jacket-found.jpg | hero_found | jaket | ponsel | 0.1298 | kartu-atm | 0.1107 | ktm | 0.0869 |
| dup-glasses-1.jpg | hero_found | kacamata | kacamata | 0.9990 | lainnya | 0.0002 | jas-almamater | 0.0002 |
| dup-glasses-2.jpg | hero_found | kacamata | ponsel | 0.1285 | dokumen | 0.0966 | ktm | 0.0882 |
| hero-flashdisk-found.jpg | hero_found | flashdisk | flashdisk | 0.9577 | power-bank | 0.0280 | charger | 0.0115 |
| hero-powerbank-found.jpg | hero_found | power-bank | laptop | 0.5501 | power-bank | 0.3134 | tablet | 0.0560 |
| hero-book-found.jpg | hero_found | buku | ponsel | 0.1496 | sim | 0.1033 | dokumen | 0.0933 |
| hero-backpack-found.jpg | hero_found | ransel | tas-laptop | 0.8815 | ransel | 0.0810 | laptop | 0.0158 |
| hero-helmet-found.jpg | hero_found | helm | helm | 0.5500 | earphone | 0.2935 | perhiasan | 0.0326 |
| bg-earphone-01.jpg | background | earphone | tablet | 0.4638 | earphone | 0.1652 | ponsel | 0.1308 |
| bg-earphone-02.jpg | background | earphone | earphone | 0.6996 | charger | 0.2355 | perhiasan | 0.0151 |
| bg-wallet-01.jpg | background | dompet | ponsel | 0.1045 | dokumen | 0.0974 | kartu-atm | 0.0905 |
| bg-wallet-02.jpg | background | dompet | pouch | 0.8647 | tote-bag | 0.0297 | dompet | 0.0192 |
| bg-keys-01.jpg | background | kunci-motor | flashdisk | 0.2062 | kunci-lain | 0.1961 | kunci-motor | 0.1829 |
| bg-keys-02.jpg | background | kunci-motor | kunci-lain | 0.5482 | kunci-motor | 0.3032 | kartu-atm | 0.0540 |
| bg-calc-01.jpg | background | kalkulator | kalkulator | 0.9984 | ponsel | 0.0003 | laptop | 0.0002 |
| bg-umbrella-01.jpg | background | payung | ponsel | 0.1292 | sim | 0.1184 | ktm | 0.0836 |
| bg-tumbler-01.jpg | background | tumbler | kartu-lain | 0.1433 | botol-minum | 0.1159 | dokumen | 0.0981 |
| bg-tumbler-02.jpg | background | tumbler | lainnya | 0.3044 | jas-almamater | 0.2986 | ponsel | 0.0971 |
| bg-jacket-01.jpg | background | jaket | dokumen | 0.1107 | kartu-atm | 0.0930 | ponsel | 0.0919 |
| bg-glasses-01.jpg | background | kacamata | kacamata | 0.9974 | lainnya | 0.0011 | perhiasan | 0.0005 |
| bg-flashdisk-01.jpg | background | flashdisk | flashdisk | 0.9577 | power-bank | 0.0280 | charger | 0.0115 |
| bg-powerbank-01.jpg | background | power-bank | laptop | 0.6877 | tablet | 0.1795 | flashdisk | 0.0644 |
| bg-book-01.jpg | background | buku | buku | 0.4500 | perhiasan | 0.2284 | alat-tulis | 0.1735 |
| bg-book-02.jpg | background | buku | buku | 0.4082 | alat-tulis | 0.2762 | lainnya | 0.0792 |
| bg-bag-01.jpg | background | ransel | sim | 0.1206 | ponsel | 0.1086 | lainnya | 0.0745 |
| bg-bag-02.jpg | background | ransel | tote-bag | 0.9924 | tas-laptop | 0.0067 | ransel | 0.0007 |
| bg-helmet-01.jpg | background | helm | sim | 0.1149 | ktm | 0.1081 | dokumen | 0.1043 |
| bg-watch-01.jpg | background | jam-tangan | jam-tangan | 0.9982 | perhiasan | 0.0013 | lainnya | 0.0002 |
| bg-watch-02.jpg | background | jam-tangan | flashdisk | 0.4702 | jam-tangan | 0.2903 | power-bank | 0.1180 |
| bg-mouse-01.jpg | background | mouse | laptop | 0.1994 | tablet | 0.1701 | jam-tangan | 0.1147 |
| bg-cable-01.jpg | background | kabel-charger | power-bank | 0.3540 | charger | 0.1770 | perhiasan | 0.0965 |
| bg-pouch-01.jpg | background | kotak-pensil | sim | 0.1369 | lainnya | 0.1030 | ponsel | 0.0900 |
| bg-shoes-01.jpg | background | sepatu | lainnya | 0.2668 | pakaian-lain | 0.2657 | earphone | 0.2556 |
| bg-hat-01.jpg | background | topi | ponsel | 0.1169 | flashdisk | 0.0975 | kartu-atm | 0.0889 |
| hero-earbuds-lost.jpg | hero_lost | earphone | charger | 0.7730 | power-bank | 0.1741 | sim | 0.0122 |
| hero-wallet-lost.jpg | hero_lost | dompet | tempat-kartu | 0.5676 | dompet | 0.4118 | kartu-atm | 0.0069 |
| hero-keys-lost.jpg | hero_lost | kunci-motor | lainnya | 0.1207 | dokumen | 0.1125 | buku | 0.0721 |
| hero-calculator-lost.jpg | hero_lost | kalkulator | kalkulator | 0.9814 | tablet | 0.0055 | laptop | 0.0042 |
| hero-umbrella-lost.jpg | hero_lost | payung | pakaian-lain | 0.5883 | lainnya | 0.1433 | pouch | 0.0798 |
| hero-tumbler-lost.jpg | hero_lost | tumbler | dokumen | 0.1385 | lainnya | 0.1234 | ponsel | 0.0747 |
| hero-jacket-lost.jpg | hero_lost | jaket | dokumen | 0.1496 | lainnya | 0.0986 | laptop | 0.0662 |
| hero-flashdisk-lost.jpg | hero_lost | flashdisk | flashdisk | 0.9726 | power-bank | 0.0216 | charger | 0.0045 |
| hero-powerbank-lost.jpg | hero_lost | power-bank | power-bank | 0.3263 | tablet | 0.2592 | laptop | 0.0836 |
| hero-book-lost.jpg | hero_lost | buku | dokumen | 0.1122 | lainnya | 0.1055 | buku | 0.0769 |
| hero-backpack-lost.jpg | hero_lost | ransel | tas-laptop | 0.8499 | ransel | 0.0974 | laptop | 0.0125 |
| hero-helmet-lost.jpg | hero_lost | helm | helm | 0.9640 | earphone | 0.0238 | lainnya | 0.0023 |

## Prompt revision 1, hashed prompt set

Scores are the live ML endpoint response for the category prompts listed in `data/seed/categories.yaml`.

| Image | Type | Expected | Top 1 | Top 1 score | Top 2 | Top 2 score | Top 3 | Top 3 score |
|---|---|---|---|---:|---|---:|---|---:|
| hero-earbuds-found.jpg | hero_found | earphone | charger | 0.8019 | power-bank | 0.1090 | sim | 0.0148 |
| hero-wallet-found.jpg | hero_found | dompet | tempat-kartu | 0.9497 | dompet | 0.0396 | pouch | 0.0065 |
| hero-keys-found.jpg | hero_found | kunci-motor | ponsel | 0.1242 | kartu-atm | 0.1047 | sim | 0.0993 |
| hero-calculator-found.jpg | hero_found | kalkulator | kalkulator | 0.9843 | tablet | 0.0066 | laptop | 0.0025 |
| hero-umbrella-found.jpg | hero_found | payung | pakaian-lain | 0.4922 | lainnya | 0.1796 | ransel | 0.0936 |
| hero-tumbler-found.jpg | hero_found | tumbler | ktm | 0.1192 | dokumen | 0.0986 | ponsel | 0.0936 |
| hero-jacket-found.jpg | hero_found | jaket | ponsel | 0.1312 | kartu-atm | 0.1119 | ktm | 0.0879 |
| dup-glasses-1.jpg | hero_found | kacamata | kacamata | 0.9989 | lainnya | 0.0002 | jas-almamater | 0.0002 |
| dup-glasses-2.jpg | hero_found | kacamata | ponsel | 0.1312 | dokumen | 0.0986 | ktm | 0.0901 |
| hero-flashdisk-found.jpg | hero_found | flashdisk | flashdisk | 0.9574 | power-bank | 0.0280 | charger | 0.0115 |
| hero-powerbank-found.jpg | hero_found | power-bank | laptop | 0.5510 | power-bank | 0.3140 | tablet | 0.0561 |
| hero-book-found.jpg | hero_found | buku | ponsel | 0.1521 | sim | 0.1050 | dokumen | 0.0948 |
| hero-backpack-found.jpg | hero_found | ransel | tas-laptop | 0.8832 | ransel | 0.0812 | laptop | 0.0159 |
| hero-helmet-found.jpg | hero_found | helm | helm | 0.7217 | perhiasan | 0.0428 | lainnya | 0.0403 |
| bg-earphone-01.jpg | background | earphone | tablet | 0.5179 | ponsel | 0.1461 | earphone | 0.0946 |
| bg-earphone-02.jpg | background | earphone | earphone | 0.7433 | charger | 0.2005 | perhiasan | 0.0129 |
| bg-wallet-01.jpg | background | dompet | ponsel | 0.1068 | dokumen | 0.0996 | kartu-atm | 0.0925 |
| bg-wallet-02.jpg | background | dompet | pouch | 0.8710 | tote-bag | 0.0299 | dompet | 0.0194 |
| bg-keys-01.jpg | background | kunci-motor | kunci-motor | 0.4995 | earphone | 0.3184 | flashdisk | 0.0556 |
| bg-keys-02.jpg | background | kunci-motor | kunci-lain | 0.6728 | kunci-motor | 0.1490 | kartu-atm | 0.0662 |
| bg-calc-01.jpg | background | kalkulator | kalkulator | 0.9984 | ponsel | 0.0003 | laptop | 0.0002 |
| bg-umbrella-01.jpg | background | payung | ponsel | 0.1310 | sim | 0.1200 | ktm | 0.0847 |
| bg-tumbler-01.jpg | background | tumbler | kartu-lain | 0.1492 | botol-minum | 0.1207 | dokumen | 0.1022 |
| bg-tumbler-02.jpg | background | tumbler | lainnya | 0.3250 | jas-almamater | 0.3187 | ponsel | 0.1037 |
| bg-jacket-01.jpg | background | jaket | dokumen | 0.1132 | kartu-atm | 0.0952 | ponsel | 0.0940 |
| bg-glasses-01.jpg | background | kacamata | kacamata | 0.9974 | lainnya | 0.0011 | perhiasan | 0.0005 |
| bg-flashdisk-01.jpg | background | flashdisk | flashdisk | 0.9574 | power-bank | 0.0280 | charger | 0.0115 |
| bg-powerbank-01.jpg | background | power-bank | laptop | 0.6875 | tablet | 0.1795 | flashdisk | 0.0644 |
| bg-book-01.jpg | background | buku | perhiasan | 0.4117 | alat-tulis | 0.3127 | lainnya | 0.1157 |
| bg-book-02.jpg | background | buku | alat-tulis | 0.4075 | buku | 0.1278 | lainnya | 0.1169 |
| bg-bag-01.jpg | background | ransel | sim | 0.1244 | ponsel | 0.1120 | lainnya | 0.0769 |
| bg-bag-02.jpg | background | ransel | tote-bag | 0.9923 | tas-laptop | 0.0067 | ransel | 0.0007 |
| bg-helmet-01.jpg | background | helm | sim | 0.1176 | ktm | 0.1106 | dokumen | 0.1067 |
| bg-watch-01.jpg | background | jam-tangan | jam-tangan | 0.9982 | perhiasan | 0.0013 | lainnya | 0.0002 |
| bg-watch-02.jpg | background | jam-tangan | flashdisk | 0.4641 | jam-tangan | 0.2866 | power-bank | 0.1165 |
| bg-mouse-01.jpg | background | mouse | laptop | 0.1858 | tablet | 0.1586 | jam-tangan | 0.1069 |
| bg-cable-01.jpg | background | kabel-charger | power-bank | 0.3793 | charger | 0.1897 | perhiasan | 0.1034 |
| bg-pouch-01.jpg | background | kotak-pensil | sim | 0.1407 | lainnya | 0.1059 | ponsel | 0.0925 |
| bg-shoes-01.jpg | background | sepatu | lainnya | 0.3577 | pakaian-lain | 0.3563 | power-bank | 0.0429 |
| bg-hat-01.jpg | background | topi | ponsel | 0.1202 | flashdisk | 0.1002 | kartu-atm | 0.0914 |
| hero-earbuds-lost.jpg | hero_lost | earphone | charger | 0.7720 | power-bank | 0.1739 | sim | 0.0122 |
| hero-wallet-lost.jpg | hero_lost | dompet | tempat-kartu | 0.5663 | dompet | 0.4108 | kartu-atm | 0.0069 |
| hero-keys-lost.jpg | hero_lost | kunci-motor | lainnya | 0.1305 | dokumen | 0.1216 | ponsel | 0.0689 |
| hero-calculator-lost.jpg | hero_lost | kalkulator | kalkulator | 0.9817 | tablet | 0.0055 | laptop | 0.0042 |
| hero-umbrella-lost.jpg | hero_lost | payung | pakaian-lain | 0.6195 | lainnya | 0.1509 | pouch | 0.0840 |
| hero-tumbler-lost.jpg | hero_lost | tumbler | dokumen | 0.1457 | lainnya | 0.1299 | ponsel | 0.0786 |
| hero-jacket-lost.jpg | hero_lost | jaket | dokumen | 0.1573 | lainnya | 0.1037 | laptop | 0.0696 |
| hero-flashdisk-lost.jpg | hero_lost | flashdisk | flashdisk | 0.9724 | power-bank | 0.0216 | charger | 0.0045 |
| hero-powerbank-lost.jpg | hero_lost | power-bank | power-bank | 0.3231 | tablet | 0.2567 | laptop | 0.0828 |
| hero-book-lost.jpg | hero_lost | buku | dokumen | 0.1218 | lainnya | 0.1145 | laptop | 0.0634 |
| hero-backpack-lost.jpg | hero_lost | ransel | tas-laptop | 0.8508 | ransel | 0.0975 | laptop | 0.0125 |
| hero-helmet-lost.jpg | hero_lost | helm | helm | 0.9833 | lainnya | 0.0024 | perhiasan | 0.0023 |\n\n

## After prompt revision and hero replacements

Scores are the live ML endpoint response for the category prompts listed in `data/seed/categories.yaml`.

| Image | Type | Expected | Top 1 | Top 1 score | Top 2 | Top 2 score | Top 3 | Top 3 score |
|---|---|---|---|---:|---|---:|---|---:|
| hero-earbuds-found.jpg | hero_found | earphone | earphone | 0.9966 | flashdisk | 0.0011 | charger | 0.0010 |
| hero-wallet-found.jpg | hero_found | dompet | tempat-kartu | 0.9497 | dompet | 0.0396 | pouch | 0.0065 |
| hero-keys-found.jpg | hero_found | kunci-motor | kunci-motor | 0.9321 | kunci-lain | 0.0288 | ransel | 0.0102 |
| hero-calculator-found.jpg | hero_found | kalkulator | kalkulator | 0.9843 | tablet | 0.0066 | laptop | 0.0025 |
| hero-umbrella-found.jpg | hero_found | payung | payung | 0.9916 | lainnya | 0.0020 | ponsel | 0.0013 |
| hero-tumbler-found.jpg | hero_found | tumbler | tumbler | 0.7502 | botol-minum | 0.2485 | lainnya | 0.0006 |
| hero-jacket-found.jpg | hero_found | jaket | jaket | 0.6630 | jas-almamater | 0.3047 | pakaian-lain | 0.0216 |
| dup-glasses-1.jpg | hero_found | kacamata | kacamata | 0.9989 | lainnya | 0.0002 | jas-almamater | 0.0002 |
| dup-glasses-2.jpg | hero_found | kacamata | kacamata | 0.9999 | perhiasan | 0.0000 | lainnya | 0.0000 |
| hero-flashdisk-found.jpg | hero_found | flashdisk | flashdisk | 0.9574 | power-bank | 0.0280 | charger | 0.0115 |
| hero-powerbank-found.jpg | hero_found | power-bank | laptop | 0.5510 | power-bank | 0.3140 | tablet | 0.0561 |
| hero-book-found.jpg | hero_found | buku | buku | 0.9103 | alat-tulis | 0.0364 | jas-almamater | 0.0202 |
| hero-backpack-found.jpg | hero_found | ransel | tas-laptop | 0.8832 | ransel | 0.0812 | laptop | 0.0159 |
| hero-helmet-found.jpg | hero_found | helm | helm | 0.7217 | perhiasan | 0.0428 | lainnya | 0.0403 |
| bg-wallet-01.jpg | background | dompet | ponsel | 0.1068 | dokumen | 0.0996 | kartu-atm | 0.0925 |
| bg-wallet-02.jpg | background | dompet | pouch | 0.8710 | tote-bag | 0.0299 | dompet | 0.0194 |
| bg-keys-01.jpg | background | kunci-motor | kunci-motor | 0.4995 | earphone | 0.3184 | flashdisk | 0.0556 |
| bg-keys-02.jpg | background | kunci-lain | kunci-lain | 0.6728 | kunci-motor | 0.1490 | kartu-atm | 0.0662 |
| bg-calc-01.jpg | background | kalkulator | kalkulator | 0.9984 | ponsel | 0.0003 | laptop | 0.0002 |
| bg-umbrella-01.jpg | background | payung | ponsel | 0.1310 | sim | 0.1200 | ktm | 0.0847 |
| bg-jacket-01.jpg | background | jaket | dokumen | 0.1132 | kartu-atm | 0.0952 | ponsel | 0.0940 |
| bg-glasses-01.jpg | background | kacamata | kacamata | 0.9974 | lainnya | 0.0011 | perhiasan | 0.0005 |
| bg-flashdisk-01.jpg | background | flashdisk | flashdisk | 0.9574 | power-bank | 0.0280 | charger | 0.0115 |
| bg-powerbank-01.jpg | background | power-bank | laptop | 0.6875 | tablet | 0.1795 | flashdisk | 0.0644 |
| bg-book-01.jpg | background | buku | perhiasan | 0.4117 | alat-tulis | 0.3127 | lainnya | 0.1157 |
| bg-book-02.jpg | background | buku | alat-tulis | 0.4075 | buku | 0.1278 | lainnya | 0.1169 |
| bg-bag-01.jpg | background | ransel | sim | 0.1244 | ponsel | 0.1120 | lainnya | 0.0769 |
| bg-bag-02.jpg | background | tote-bag | tote-bag | 0.9923 | tas-laptop | 0.0067 | ransel | 0.0007 |
| bg-helmet-01.jpg | background | helm | sim | 0.1176 | ktm | 0.1106 | dokumen | 0.1067 |
| bg-watch-01.jpg | background | jam-tangan | jam-tangan | 0.9982 | perhiasan | 0.0013 | lainnya | 0.0002 |
| bg-watch-02.jpg | background | jam-tangan | flashdisk | 0.4641 | jam-tangan | 0.2866 | power-bank | 0.1165 |
| bg-mouse-01.jpg | background | mouse | laptop | 0.1858 | tablet | 0.1586 | jam-tangan | 0.1069 |
| bg-pouch-01.jpg | background | kotak-pensil | sim | 0.1407 | lainnya | 0.1059 | ponsel | 0.0925 |
| bg-shoes-01.jpg | background | sepatu | lainnya | 0.3577 | pakaian-lain | 0.3563 | power-bank | 0.0429 |
| bg-hat-01.jpg | background | topi | ponsel | 0.1202 | flashdisk | 0.1002 | kartu-atm | 0.0914 |
| bg-earphone-003.jpg | background | earphone | earphone | 0.9391 | charger | 0.0263 | power-bank | 0.0185 |
| bg-earphone-005.jpg | background | earphone | kunci-motor | 0.6890 | earphone | 0.1074 | flashdisk | 0.0639 |
| bg-earphone-006.jpg | background | earphone | earphone | 0.9995 | charger | 0.0001 | perhiasan | 0.0001 |
| bg-dompet-001.jpg | background | dompet | dompet | 0.7216 | tempat-kartu | 0.2643 | pouch | 0.0124 |
| bg-dompet-002.jpg | background | dompet | dompet | 0.6357 | tempat-kartu | 0.3017 | earphone | 0.0343 |
| bg-kalkulator-001.jpg | background | kalkulator | kalkulator | 0.9923 | lainnya | 0.0036 | ponsel | 0.0017 |
| bg-kalkulator-002.jpg | background | kalkulator | kalkulator | 0.9984 | laptop | 0.0004 | tablet | 0.0002 |
| bg-kalkulator-003.jpg | background | kalkulator | kalkulator | 0.9211 | jam-tangan | 0.0331 | tablet | 0.0159 |
| bg-kalkulator-004.jpg | background | kalkulator | kalkulator | 0.8645 | tablet | 0.0391 | laptop | 0.0282 |
| bg-payung-001.jpg | background | payung | payung | 0.9916 | lainnya | 0.0020 | ponsel | 0.0013 |
| bg-payung-002.jpg | background | payung | payung | 0.6473 | tote-bag | 0.0889 | topi | 0.0434 |
| bg-payung-003.jpg | background | payung | perhiasan | 0.5820 | kunci-lain | 0.3189 | kacamata | 0.0601 |
| bg-jaket-001.jpg | background | jaket | pakaian-lain | 0.6389 | tote-bag | 0.1518 | jaket | 0.1244 |
| bg-kacamata-001.jpg | background | kacamata | kacamata | 0.9985 | charger | 0.0002 | laptop | 0.0002 |
| bg-flashdisk-002.jpg | background | flashdisk | flashdisk | 0.3003 | power-bank | 0.1544 | kartu-atm | 0.1069 |
| bg-flashdisk-003.jpg | background | flashdisk | flashdisk | 0.6016 | tablet | 0.0923 | laptop | 0.0677 |
| bg-power-bank-001.jpg | background | power-bank | power-bank | 0.8124 | charger | 0.0834 | laptop | 0.0579 |
| bg-power-bank-003.jpg | background | power-bank | laptop | 0.2917 | power-bank | 0.2659 | flashdisk | 0.2605 |
| bg-buku-001.jpg | background | buku | buku | 0.9103 | alat-tulis | 0.0364 | jas-almamater | 0.0202 |
| bg-buku-002.jpg | background | buku | tote-bag | 0.7767 | buku | 0.0795 | alat-tulis | 0.0469 |
| bg-buku-004.jpg | background | buku | buku | 0.7480 | alat-tulis | 0.1868 | lainnya | 0.0192 |
| bg-helm-004.jpg | background | helm | kacamata | 0.4790 | helm | 0.4475 | perhiasan | 0.0221 |
| bg-jam-tangan-001.jpg | background | jam-tangan | jam-tangan | 0.9963 | perhiasan | 0.0032 | lainnya | 0.0002 |
| bg-jam-tangan-002.jpg | background | jam-tangan | jam-tangan | 0.9967 | flashdisk | 0.0011 | earphone | 0.0010 |
| bg-jam-tangan-003.jpg | background | jam-tangan | jam-tangan | 0.9995 | perhiasan | 0.0003 | lainnya | 0.0001 |
| bg-mouse-002.jpg | background | mouse | helm | 0.8170 | kunci-motor | 0.0360 | earphone | 0.0343 |
| bg-mouse-003.jpg | background | mouse | flashdisk | 0.3555 | charger | 0.2663 | kunci-motor | 0.1789 |
| bg-mouse-004.jpg | background | mouse | flashdisk | 0.3120 | laptop | 0.2053 | tablet | 0.1792 |
| bg-earphone-007.jpg | background | earphone | earphone | 0.9966 | flashdisk | 0.0011 | charger | 0.0010 |
| bg-earphone-010.jpg | background | earphone | earphone | 0.9985 | charger | 0.0006 | power-bank | 0.0005 |
| bg-dompet-003.jpg | background | dompet | tempat-kartu | 0.6871 | dompet | 0.3108 | pouch | 0.0012 |
| bg-dompet-004.jpg | background | dompet | dompet | 0.7478 | tempat-kartu | 0.2079 | pouch | 0.0423 |
| bg-kalkulator-005.jpg | background | kalkulator | kalkulator | 0.9991 | ponsel | 0.0005 | lainnya | 0.0002 |
| bg-kalkulator-006.jpg | background | kalkulator | kalkulator | 0.9777 | ponsel | 0.0044 | power-bank | 0.0043 |
| bg-kalkulator-007.jpg | background | kalkulator | kalkulator | 0.9852 | jam-tangan | 0.0035 | tablet | 0.0028 |
| bg-kalkulator-008.jpg | background | kalkulator | kalkulator | 0.6575 | tablet | 0.2121 | laptop | 0.0854 |
| bg-kacamata-005.jpg | background | kacamata | kacamata | 0.9996 | lainnya | 0.0001 | jas-almamater | 0.0001 |
| bg-kacamata-007.jpg | background | kacamata | kacamata | 0.8843 | lainnya | 0.0302 | buku | 0.0231 |
| bg-kacamata-008.jpg | background | kacamata | kacamata | 0.9991 | lainnya | 0.0004 | buku | 0.0002 |
| bg-flashdisk-004.jpg | background | flashdisk | flashdisk | 0.8359 | power-bank | 0.0459 | kartu-lain | 0.0284 |
| bg-flashdisk-005.jpg | background | flashdisk | flashdisk | 0.7282 | power-bank | 0.1335 | tablet | 0.0376 |
| bg-power-bank-005.jpg | background | power-bank | charger | 0.5743 | flashdisk | 0.2111 | power-bank | 0.1671 |
| bg-power-bank-006.jpg | background | power-bank | flashdisk | 0.2470 | kunci-lain | 0.1657 | charger | 0.1519 |
| bg-power-bank-007.jpg | background | power-bank | tempat-kartu | 0.4828 | kartu-atm | 0.2780 | dompet | 0.1310 |
| bg-buku-005.jpg | background | buku | pouch | 0.3677 | tempat-kartu | 0.1788 | dompet | 0.0899 |
| bg-buku-006.jpg | background | buku | laptop | 0.4926 | alat-tulis | 0.2041 | flashdisk | 0.0819 |
| bg-buku-007.jpg | background | buku | buku | 0.2880 | kalkulator | 0.2195 | ransel | 0.1880 |
| bg-helm-006.jpg | background | helm | topi | 0.8692 | perhiasan | 0.0339 | helm | 0.0215 |
| bg-helm-007.jpg | background | helm | helm | 0.5036 | kacamata | 0.1634 | buku | 0.0558 |
| bg-jam-tangan-005.jpg | background | jam-tangan | jam-tangan | 0.9903 | perhiasan | 0.0053 | kacamata | 0.0008 |
| bg-jam-tangan-006.jpg | background | jam-tangan | jam-tangan | 0.9162 | perhiasan | 0.0327 | ponsel | 0.0314 |
| bg-jam-tangan-007.jpg | background | jam-tangan | jam-tangan | 0.9991 | perhiasan | 0.0005 | jas-almamater | 0.0002 |
| bg-mouse-005.jpg | background | mouse | earphone | 0.3108 | alat-tulis | 0.2009 | dokumen | 0.1055 |
| bg-mouse-006.jpg | background | mouse | charger | 0.7629 | earphone | 0.0952 | power-bank | 0.0513 |
| bg-kabel-charger-001.jpg | background | kabel-charger | charger | 0.7058 | earphone | 0.1465 | laptop | 0.0327 |
| bg-kabel-charger-003.jpg | background | kabel-charger | charger | 0.4986 | flashdisk | 0.1373 | power-bank | 0.0789 |
| bg-kotak-pensil-001.jpg | background | kotak-pensil | dompet | 0.4493 | pouch | 0.4098 | tempat-kartu | 0.0671 |
| bg-sepatu-001.jpg | background | sepatu | lainnya | 0.4654 | jas-almamater | 0.4018 | pakaian-lain | 0.0359 |
| bg-sepatu-002.jpg | background | sepatu | lainnya | 0.4557 | pakaian-lain | 0.1700 | jas-almamater | 0.1192 |
| bg-earphone-011.jpg | background | earphone | power-bank | 0.4768 | charger | 0.3054 | laptop | 0.1273 |
| bg-earphone-012.jpg | background | earphone | earphone | 0.7230 | charger | 0.0859 | perhiasan | 0.0747 |
| bg-earphone-014.jpg | background | earphone | earphone | 0.9857 | charger | 0.0078 | power-bank | 0.0022 |
| bg-dompet-005.jpg | background | dompet | dompet | 0.4060 | pouch | 0.2979 | tempat-kartu | 0.1211 |
| bg-dompet-006.jpg | background | dompet | tas-laptop | 0.6559 | tote-bag | 0.1379 | pouch | 0.0811 |
| bg-kunci-motor-001.jpg | background | kunci-motor | kunci-motor | 0.9321 | kunci-lain | 0.0288 | ransel | 0.0102 |
| bg-kalkulator-009.jpg | background | kalkulator | ponsel | 0.7150 | kalkulator | 0.0839 | tempat-kartu | 0.0757 |
| bg-kalkulator-010.jpg | background | kalkulator | kalkulator | 0.9666 | ponsel | 0.0213 | sim | 0.0034 |
| bg-kalkulator-011.jpg | background | kalkulator | kalkulator | 0.9413 | ponsel | 0.0381 | jam-tangan | 0.0026 |
| bg-kalkulator-012.jpg | background | kalkulator | kalkulator | 0.9963 | laptop | 0.0020 | tablet | 0.0008 |
| bg-payung-005.jpg | background | payung | perhiasan | 0.5446 | kunci-lain | 0.3979 | flashdisk | 0.0272 |
| bg-jaket-002.jpg | background | jaket | jaket | 0.6630 | jas-almamater | 0.3047 | pakaian-lain | 0.0216 |
| bg-kacamata-009.jpg | background | kacamata | kacamata | 0.9945 | perhiasan | 0.0040 | earphone | 0.0003 |
| bg-kacamata-012.jpg | background | kacamata | kacamata | 0.9999 | perhiasan | 0.0000 | lainnya | 0.0000 |
| bg-flashdisk-006.jpg | background | flashdisk | flashdisk | 0.8815 | power-bank | 0.0309 | kartu-atm | 0.0213 |
| bg-flashdisk-007.jpg | background | flashdisk | flashdisk | 0.9231 | kartu-atm | 0.0171 | power-bank | 0.0129 |
| bg-power-bank-008.jpg | background | power-bank | power-bank | 0.8687 | flashdisk | 0.0811 | charger | 0.0303 |
| bg-power-bank-010.jpg | background | power-bank | power-bank | 0.9458 | flashdisk | 0.0403 | charger | 0.0070 |
| bg-buku-009.jpg | background | buku | laptop | 0.8036 | tablet | 0.0956 | lainnya | 0.0343 |
| bg-buku-010.jpg | background | buku | buku | 0.7569 | lainnya | 0.2110 | jas-almamater | 0.0099 |
| bg-helm-008.jpg | background | helm | helm | 0.9964 | topi | 0.0009 | kunci-motor | 0.0006 |
| bg-helm-009.jpg | background | helm | topi | 0.9472 | helm | 0.0201 | pakaian-lain | 0.0054 |
| bg-jam-tangan-008.jpg | background | jam-tangan | jam-tangan | 0.8660 | perhiasan | 0.0553 | helm | 0.0220 |
| bg-jam-tangan-009.jpg | background | jam-tangan | jam-tangan | 0.9604 | ponsel | 0.0188 | perhiasan | 0.0111 |
| bg-jam-tangan-010.jpg | background | jam-tangan | jam-tangan | 0.9364 | earphone | 0.0237 | kunci-motor | 0.0115 |
| bg-mouse-007.jpg | background | mouse | ransel | 0.1907 | laptop | 0.1784 | tablet | 0.1249 |
| bg-mouse-009.jpg | background | mouse | laptop | 0.6394 | tablet | 0.2844 | kalkulator | 0.0252 |
| bg-kabel-charger-004.jpg | background | kabel-charger | charger | 0.3456 | flashdisk | 0.2309 | power-bank | 0.1607 |
| bg-kabel-charger-005.jpg | background | kabel-charger | power-bank | 0.8251 | laptop | 0.0368 | tas-laptop | 0.0297 |
| bg-mouse-010.jpg | background | mouse | earphone | 0.9632 | jam-tangan | 0.0068 | flashdisk | 0.0049 |
| bg-mouse-011.jpg | background | mouse | tempat-kartu | 0.3459 | tablet | 0.2384 | kartu-lain | 0.1438 |
| bg-sepatu-003.jpg | background | sepatu | pakaian-lain | 0.5695 | perhiasan | 0.0897 | jas-almamater | 0.0815 |
| bg-sepatu-004.jpg | background | sepatu | lainnya | 0.4032 | pakaian-lain | 0.2525 | topi | 0.0442 |
| bg-sepatu-005.jpg | background | sepatu | pouch | 0.3396 | tote-bag | 0.2087 | jaket | 0.1771 |
| bg-payung-006.jpg | background | payung | payung | 0.8275 | buku | 0.0903 | tote-bag | 0.0184 |
| bg-earphone-015.jpg | background | earphone | earphone | 0.7255 | flashdisk | 0.1256 | power-bank | 0.1115 |
| bg-kalkulator-013.jpg | background | kalkulator | dompet | 0.8776 | tempat-kartu | 0.0886 | pouch | 0.0283 |
| bg-kalkulator-014.jpg | background | kalkulator | tablet | 0.3424 | kalkulator | 0.3392 | kacamata | 0.1940 |
| bg-kacamata-013.jpg | background | kacamata | kacamata | 0.7585 | perhiasan | 0.1142 | kunci-motor | 0.0421 |
| bg-flashdisk-008.jpg | background | flashdisk | flashdisk | 0.9794 | charger | 0.0081 | power-bank | 0.0063 |
| bg-jam-tangan-011.jpg | background | jam-tangan | jam-tangan | 0.9880 | perhiasan | 0.0105 | kacamata | 0.0004 |
| bg-jam-tangan-013.jpg | background | jam-tangan | jam-tangan | 0.9966 | perhiasan | 0.0023 | lainnya | 0.0007 |
| bg-flashdisk-009.jpg | background | flashdisk | flashdisk | 0.5193 | power-bank | 0.3919 | ponsel | 0.0329 |
| bg-flashdisk-010.jpg | background | flashdisk | flashdisk | 0.8863 | kunci-motor | 0.0503 | charger | 0.0316 |
| bg-jam-tangan-014.jpg | background | jam-tangan | jam-tangan | 0.9994 | perhiasan | 0.0004 | lainnya | 0.0001 |
| bg-mouse-016.jpg | background | mouse | charger | 0.3834 | flashdisk | 0.1566 | kunci-lain | 0.1110 |
| bg-mouse-018.jpg | background | mouse | laptop | 0.4663 | tablet | 0.2372 | alat-tulis | 0.0652 |
| bg-sepatu-006.jpg | background | sepatu | pakaian-lain | 0.6168 | lainnya | 0.1497 | kacamata | 0.0732 |
| bg-flashdisk-011.jpg | background | flashdisk | earphone | 0.8381 | flashdisk | 0.0562 | kunci-lain | 0.0242 |
| bg-power-bank-013.jpg | background | power-bank | tempat-kartu | 0.4269 | power-bank | 0.3769 | flashdisk | 0.1410 |
| bg-buku-013.jpg | background | buku | buku | 0.9973 | alat-tulis | 0.0016 | dokumen | 0.0003 |
| bg-jam-tangan-015.jpg | background | jam-tangan | jam-tangan | 0.8794 | perhiasan | 0.1035 | lainnya | 0.0021 |
| bg-tumbler-hero-candidate.jpg | background | tumbler | tumbler | 0.7502 | botol-minum | 0.2485 | lainnya | 0.0006 |
| hero-earbuds-lost.jpg | hero_lost | earphone | earphone | 0.9976 | flashdisk | 0.0010 | charger | 0.0006 |
| hero-wallet-lost.jpg | hero_lost | dompet | tempat-kartu | 0.5663 | dompet | 0.4108 | kartu-atm | 0.0069 |
| hero-keys-lost.jpg | hero_lost | kunci-motor | kunci-motor | 0.9846 | kunci-lain | 0.0032 | charger | 0.0016 |
| hero-calculator-lost.jpg | hero_lost | kalkulator | kalkulator | 0.9817 | tablet | 0.0055 | laptop | 0.0042 |
| hero-umbrella-lost.jpg | hero_lost | payung | payung | 0.9956 | lainnya | 0.0007 | power-bank | 0.0007 |
| hero-tumbler-lost.jpg | hero_lost | tumbler | tumbler | 0.8043 | botol-minum | 0.1944 | lainnya | 0.0005 |
| hero-jacket-lost.jpg | hero_lost | jaket | jaket | 0.7047 | jas-almamater | 0.2866 | pakaian-lain | 0.0057 |
| hero-flashdisk-lost.jpg | hero_lost | flashdisk | flashdisk | 0.9724 | power-bank | 0.0216 | charger | 0.0045 |
| hero-powerbank-lost.jpg | hero_lost | power-bank | power-bank | 0.3231 | tablet | 0.2567 | laptop | 0.0828 |
| hero-book-lost.jpg | hero_lost | buku | buku | 0.8917 | lainnya | 0.0434 | alat-tulis | 0.0218 |
| hero-backpack-lost.jpg | hero_lost | ransel | tas-laptop | 0.8508 | ransel | 0.0975 | laptop | 0.0125 |
| hero-helmet-lost.jpg | hero_lost | helm | helm | 0.9833 | lainnya | 0.0024 | perhiasan | 0.0023 |


## Live lost-report holdout

The reserved g-flashdisk-011.jpg image was uploaded through the member lost-report form. The live web assistant returned Earphone at 0.8996, Flashdisk at 0.0327, and Kunci Lain at 0.0139. The report was manually corrected to Flashdisk. The separate CLI image-model result above is not interchangeable with the web assistant result because the endpoints use different inference paths. This single holdout is a workflow check, not an accuracy estimate.

## T1 before alternative hero selection

Measured on 21 October 2026 using `inspect_model_photos.py` against the resident real
models. Before replacement, 11/14 heroes have the expected Top 1 category and 14/14
include it in Top 3. The fixed demo sample is not an independent accuracy evaluation.

Scores are the live ML endpoint response for the category prompts listed in `data/seed/categories.yaml`.

| Image | Type | Expected | Top 1 | Top 1 score | Top 2 | Top 2 score | Top 3 | Top 3 score |
|---|---|---|---|---:|---|---:|---|---:|
| hero-earbuds-found.jpg | hero_found | earphone | earphone | 0.9966 | flashdisk | 0.0011 | charger | 0.0010 |
| hero-wallet-found.jpg | hero_found | dompet | tempat-kartu | 0.9497 | dompet | 0.0396 | pouch | 0.0065 |
| hero-keys-found.jpg | hero_found | kunci-motor | kunci-motor | 0.9321 | kunci-lain | 0.0288 | ransel | 0.0102 |
| hero-calculator-found.jpg | hero_found | kalkulator | kalkulator | 0.9843 | tablet | 0.0066 | laptop | 0.0025 |
| hero-umbrella-found.jpg | hero_found | payung | payung | 0.9916 | lainnya | 0.0020 | ponsel | 0.0013 |
| hero-tumbler-found.jpg | hero_found | tumbler | tumbler | 0.7502 | botol-minum | 0.2485 | lainnya | 0.0006 |
| hero-jacket-found.jpg | hero_found | jaket | jaket | 0.6630 | jas-almamater | 0.3047 | pakaian-lain | 0.0216 |
| dup-glasses-1.jpg | hero_found | kacamata | kacamata | 0.9989 | lainnya | 0.0002 | jas-almamater | 0.0002 |
| dup-glasses-2.jpg | hero_found | kacamata | kacamata | 0.9999 | perhiasan | 0.0000 | lainnya | 0.0000 |
| hero-flashdisk-found.jpg | hero_found | flashdisk | flashdisk | 0.9574 | power-bank | 0.0280 | charger | 0.0115 |
| hero-powerbank-found.jpg | hero_found | power-bank | laptop | 0.5510 | power-bank | 0.3140 | tablet | 0.0561 |
| hero-book-found.jpg | hero_found | buku | buku | 0.9103 | alat-tulis | 0.0364 | jas-almamater | 0.0202 |
| hero-backpack-found.jpg | hero_found | ransel | tas-laptop | 0.8832 | ransel | 0.0812 | laptop | 0.0159 |
| hero-helmet-found.jpg | hero_found | helm | helm | 0.7217 | perhiasan | 0.0428 | lainnya | 0.0403 |

## T1 alternative photos with unchanged prompts

Three visually reviewed Commons photographs per category were tested with the same
prompt hash. Candidate assets remain attributed in `SOURCES.csv` as `hero_candidate`
and are excluded from report seeding. Photos with people, illustrations or visible
personal cards were rejected before model analysis and are not stored in the pack.

Scores are the live ML endpoint response for the category prompts listed in `data/seed/categories.yaml`.

| Image | Type | Expected | Top 1 | Top 1 score | Top 2 | Top 2 score | Top 3 | Top 3 score |
|---|---|---|---|---:|---|---:|---|---:|
| t1-dompet-1.jpg | hero_candidate | dompet | dompet | 0.7216 | tempat-kartu | 0.2643 | pouch | 0.0124 |
| t1-power-bank-1.jpg | hero_candidate | power-bank | tempat-kartu | 0.4828 | kartu-atm | 0.2780 | dompet | 0.1310 |
| t1-power-bank-2.jpg | hero_candidate | power-bank | power-bank | 0.5279 | flashdisk | 0.2122 | tempat-kartu | 0.0778 |
| t1-power-bank-3.jpg | hero_candidate | power-bank | power-bank | 0.8124 | charger | 0.0834 | laptop | 0.0579 |
| t1-ransel-2.jpg | hero_candidate | ransel | tas-laptop | 0.6089 | ransel | 0.3628 | tote-bag | 0.0194 |
| t1-ransel-4.jpg | hero_candidate | ransel | ransel | 0.8095 | tas-laptop | 0.1575 | jas-almamater | 0.0101 |
| t1-ransel-5.jpg | hero_candidate | ransel | ransel | 0.5808 | tas-laptop | 0.2323 | jas-almamater | 0.1014 |
| t1-dompet-4.jpg | hero_candidate | dompet | dompet | 0.5774 | tempat-kartu | 0.3496 | pouch | 0.0559 |
| t1-dompet-5.jpg | hero_candidate | dompet | dompet | 0.7547 | tempat-kartu | 0.2389 | pouch | 0.0033 |

## T1 final heroes, unchanged category prompts

Selected `t1-dompet-1.jpg`, `t1-power-bank-3.jpg` and `t1-ransel-5.jpg`: a closed
leather billfold, a portable battery with USB ports visible, and a school backpack
against a plain wall. All are plausible campus lost items. The backpack was chosen
for its unobstructed school-bag view rather than the highest candidate score.
Their lost-photo pairs use the documented crop and lighting transform. Category
prompts were unchanged. Top 1 improves to 14/14 (100%); Top 3 remains 14/14 (100%).
These are image-endpoint rankings, not a guarantee of the web assistant's output.

Scores are the live ML endpoint response for the category prompts listed in `data/seed/categories.yaml`.

| Image | Type | Expected | Top 1 | Top 1 score | Top 2 | Top 2 score | Top 3 | Top 3 score |
|---|---|---|---|---:|---|---:|---|---:|
| hero-earbuds-found.jpg | hero_found | earphone | earphone | 0.9966 | flashdisk | 0.0011 | charger | 0.0010 |
| hero-wallet-found.jpg | hero_found | dompet | dompet | 0.7216 | tempat-kartu | 0.2643 | pouch | 0.0124 |
| hero-keys-found.jpg | hero_found | kunci-motor | kunci-motor | 0.9321 | kunci-lain | 0.0288 | ransel | 0.0102 |
| hero-calculator-found.jpg | hero_found | kalkulator | kalkulator | 0.9843 | tablet | 0.0066 | laptop | 0.0025 |
| hero-umbrella-found.jpg | hero_found | payung | payung | 0.9916 | lainnya | 0.0020 | ponsel | 0.0013 |
| hero-tumbler-found.jpg | hero_found | tumbler | tumbler | 0.7502 | botol-minum | 0.2485 | lainnya | 0.0006 |
| hero-jacket-found.jpg | hero_found | jaket | jaket | 0.6630 | jas-almamater | 0.3047 | pakaian-lain | 0.0216 |
| dup-glasses-1.jpg | hero_found | kacamata | kacamata | 0.9989 | lainnya | 0.0002 | jas-almamater | 0.0002 |
| dup-glasses-2.jpg | hero_found | kacamata | kacamata | 0.9999 | perhiasan | 0.0000 | lainnya | 0.0000 |
| hero-flashdisk-found.jpg | hero_found | flashdisk | flashdisk | 0.9574 | power-bank | 0.0280 | charger | 0.0115 |
| hero-powerbank-found.jpg | hero_found | power-bank | power-bank | 0.8124 | charger | 0.0834 | laptop | 0.0579 |
| hero-book-found.jpg | hero_found | buku | buku | 0.9103 | alat-tulis | 0.0364 | jas-almamater | 0.0202 |
| hero-backpack-found.jpg | hero_found | ransel | ransel | 0.5808 | tas-laptop | 0.2323 | jas-almamater | 0.1014 |
| hero-helmet-found.jpg | hero_found | helm | helm | 0.7217 | perhiasan | 0.0428 | lainnya | 0.0403 |
