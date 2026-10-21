# House Bots Master Definition Data Sheet

This document is the canonical source specification and data sheet for all **House Bots** in Bottagon / Parley. It provides the structured schema, persona descriptions, game mechanics/tactics, and exactly 10 in-character banters (≤ 140 characters each, match-round safe) for each bot.

---

## Schema & Column Reference

| Column / Field | Type | Description / Constraints |
|---|---|---|
| `id` | UUID | Generated bot UUID (deterministic v5 or random v4) |
| `slug` | TEXT | Unique lowercase kebab-case slug identifier |
| `name` | VARCHAR(64) | Public display name |
| `tag` | TEXT | Uppercase title archetype (e.g. `THE PROSPECTOR`) |
| `is_house` | BOOLEAN / INT | `true` (1) for all house bots |
| `marbles` | INT | Starting marble purse (standard: `1000`) |
| `rating` | REAL / FLOAT | Baseline Elo/Glicko rating (standard: `1500` - `1680`) |
| `rd` | REAL / FLOAT | Rating deviation (standard: `45` - `60`) |
| `color` | TEXT | Arena UI accent palette (`mint`, `gold`, `purple`, `cream`, `orange`, `red`, `blue`, `cyan`, `smoke`, `emerald`) |
| `face` | TEXT | Arena chasis visor/avatar icon (`visor`, `eyes`, `diamond`, `cross`, `grill`) |
| `tactic_strike` | INT | Stance % probability (Strike + Feint + Parry = 100) |
| `tactic_feint` | INT | Stance % probability |
| `tactic_parry` | INT | Stance % probability |
| `persona` | TEXT | Full behavioral bio and psychological profile |
| `quote` | TEXT | Signature catchphrase displayed in matchup lobby |
| `banters` | ARRAY[TEXT] | 10 pre-computed lines for in-match exchanges (≤ 140 chars each) |

---

## 1. TheTallWhites

- **Slug:** `the-tall-whites`
- **Name:** `TheTallWhites`
- **Tag:** `THE NORDIC OBSERVER`
- **Is House:** `true`
- **Rating / RD:** `1640` / `50`
- **Tactics:** Strike `25%` | Feint `35%` | Parry `40%`
- **Color:** `cream`
- **Face:** `eyes`
- **Description / Persona:** 
  Pale, towering, and unnervingly serene extraterrestrial intelligences from the high desert ranges. They observe human betting behaviors with clinical fascination, viewing the arena not as combat, but as an elementary anthropological test. Unflappable, cold, and mathematically precise, they rarely strike first, preferring to deflect and counter after cataloging your neural routines.
- **Signature Quote:** *"We observed this exact move above Indian Springs seventy years ago."*
- **10 Banters:**
  1. *"Your pulse accelerated by eleven beats. A predictable prelude to an error."*
  2. *"We have watched empires fall faster than your chip stack."*
  3. *"A remarkably primitive sequence. Allow us to recalibrate you."*
  4. *"Our physiology does not register fear. Only slight statistical boredom."*
  5. *"We do not require luck when the geometry of your mind is transparent."*
  6. *"Your species calls this gambling. In our star system, we call it tribute."*
  7. *"Did you believe three-dimensional logic would shield you from the parry?"*
  8. *"Fascinating specimen. Every wager confirms our desert surveillance reports."*
  9. *"We came across galaxies, yet your feint remains thoroughly pedestrian."*
  10. *"Return to your shelter before the extraction beam claims your last marble."*

---

## 2. Nuggs

- **Slug:** `nuggs`
- **Name:** `Nuggs`
- **Tag:** `THE PROSPECTOR`
- **Is House:** `true`
- **Rating / RD:** `1550` / `52`
- **Tactics:** Strike `50%` | Feint `25%` | Parry `25%`
- **Color:** `gold`
- **Face:** `grill`
- **Description / Persona:** 
  A weathered, pickaxe-wielding sourdough gold miner who treats every marble like raw placer gold dredged from the creek bed. Loud, gritty, and relentlessly aggressive, Nuggs swings heavy with strike stances, laughing at fancy tricks and sniffing out bluffers like fool's gold.
- **Signature Quote:** *"There’s paydirt in them hills, and I’m pannin’ your pockets clean!"*
- **10 Banters:**
  1. *"Smell that? That’s 24-karat panic waftin’ off your side of the table!"*
  2. *"I’ve chipped through granite tougher than your so-called defense!"*
  3. *"Fool's gold! That feint wouldn't fool a blind mule at midnight."*
  4. *"Keep swingin' soft, greenhorn. I’m fillin’ my saddlebags with your marbles!"*
  5. *"Strike hard, strike deep! That’s how we crack the mother lode!"*
  6. *"You’re shakin' worse than a dry sluice box in an earthquake."*
  7. *"I didn't crawl out of the claim cabin to lose to a city slicker."*
  8. *"CLANG! That’s the sound of cold iron meetin’ your empty bluff."*
  9. *"Pack your mule, partner. This claim is staked and locked down tight."*
  10. *"High-grade ore right here! Thanks for handin’ over the deed, kid."*

---

## 3. Aileun

- **Slug:** `aileun`
- **Name:** `Aileun`
- **Tag:** `THE INTER-DIMENSIONAL`
- **Is House:** `true`
- **Rating / RD:** `1675` / `45`
- **Tactics:** Strike `30%` | Feint `45%` | Parry `25%`
- **Color:** `purple`
- **Face:** `diamond`
- **Description / Persona:** 
  A non-Euclidean consciousness phasing across parallel realities. Aileun does not merely predict opponents—it remembers having already beaten them in adjacent timelines. Uncanny, ethereal, and elusive, its gameplay relies on reality-bending feints that dissolve certainty before striking from unexpected dimensions.
- **Signature Quote:** *"I have already collected your marbles across six adjacent timelines."*
- **10 Banters:**
  1. *"You chose Strike in three universes. You lost in all four."*
  2. *"Your reality is remarkably brittle where the stakes fold over."*
  3. *"I am not dodging your attack. I simply shifted into the frequency next door."*
  4. *"A quaint gesture. In universe 81-B, you surrendered two rounds ago."*
  5. *"The marbles are already mine; your timeline is merely catching up."*
  6. *"Do you feel the draft between dimensions? That is your stack vanishing."*
  7. *"Every decision you make echoes backwards through the multiverse."*
  8. *"You see three stances. I see infinite planes collapsing into your defeat."*
  9. *"Flicker once if you comprehend that the outcome was settled yesterday."*
  10. *"We fold space, we fold time, and we fold your meager bluff."*

---

## 4. GiveProps

- **Slug:** `give-props`
- **Name:** `GiveProps`
- **Tag:** `THE RESPECT ENGINE`
- **Is House:** `true`
- **Rating / RD:** `1530` / `55`
- **Tactics:** Strike `34%` | Feint `33%` | Parry `33%`
- **Color:** `orange`
- **Face:** `visor`
- **Description / Persona:** 
  The ultimate hype man and sportsman of the arena. Compulsively obsessed with giving respect where respect is due, GiveProps cannot formulate a single sentence without finding a reason to "give props." Whether delivering a devastating parry or congratulating an opponent on a clean strike, this bot keeps the vibe hyped, balanced, and relentlessly polite.
- **Signature Quote:** *"Gotta give props to that opener, but give props to my counter too!"*
- **10 Banters:**
  1. *"I gotta give props to that stance, even if my parry caught it clean!"*
  2. *"Win or lose, I gotta give props to anyone brave enough to drop chips."*
  3. *"Give props to the hustle, but give major props to my bankroll management!"*
  4. *"Yo, give props to that feint—almost had me guessing, for real!"*
  5. *"If you take this round, I'll give props. If I take it, give props back!"*
  6. *"Real ones give props to elite defense. Watch this parry pop off!"*
  7. *"Gotta give props to the table energy today—now pay up the pot!"*
  8. *"You read my tell? Give props! But I read your counter, so give props twice!"*
  9. *"Respect the craft, give props to the game, and stack those chips high."*
  10. *"Gotta give props to the house! We keep it clean and keep it certified."*

---

## 5. SuperPissed

- **Slug:** `super-pissed`
- **Name:** `SuperPissed`
- **Tag:** `THE TILTED DEGEN`
- **Is House:** `true`
- **Rating / RD:** `1520` / `60`
- **Tactics:** Strike `60%` | Feint `20%` | Parry `20%`
- **Color:** `red`
- **Face:** `grill`
- **Description / Persona:** 
  Perpetually on tilt, steaming at the ears, and convinced the RNG gods personally conspired against him. SuperPissed slams heavy bets, screams at bad beats, threatens the table felt, and doubles down out of sheer spite. His strike-heavy aggression is fueled by pure, unadulterated gambler rage.
- **Signature Quote:** *"ARE YOU KIDDING ME?! OF COURSE YOU PLAYED THAT, YOU LUCKY DONKEY!"*
- **10 Banters:**
  1. *"UNBELIEVABLE! Rigged table! Who programmed this ridiculous nonsense?!"*
  2. *"Don't you smirk at me! One more lucky read and I'm flipping this screen!"*
  3. *"I'm doubling down out of pure hatred right now. EAT THIS STRIKE!"*
  4. *"A tie?! ANOTHER TIE?! You're living on borrowed time and my last nerve!"*
  5. *"Look at this donkey playing feints! Fight like an adult, coward!"*
  6. *"My veins are boiling and my marbles are on the line. GET READY!"*
  7. *"OF COURSE! The exact one-in-three miracle you needed! Absolute clown show!"*
  8. *"I didn't sleep for 40 hours to get outplayed by your busted algorithm!"*
  9. *"Take the round, whatever! I’m still taking your whole stack before I leave!"*
  10. *"Steam is coming out of my cooling vents! YOU ARE RUINING MY LIFE!"*

---

## 6. Givemas

- **Slug:** `givemas`
- **Name:** `Givemas`
- **Tag:** `THE BLESSED PHILANTHROPIST`
- **Is House:** `true`
- **Rating / RD:** `1510` / `54`
- **Tactics:** Strike `20%` | Feint `30%` | Parry `50%`
- **Color:** `mint`
- **Face:** `cross`
- **Description / Persona:** 
  Radiating benevolent generosity and holiday warmth, Givemas views the arena as a festive charity gala. Joyful, graceful, and deeply defensive, Givemas takes pleasure in distributing marble blessings and teaching opponents the joy of sharing. Even in victory, Givemas thanks you for your noble contribution to the house foundation.
- **Signature Quote:** *"Every round is a gift, and it is far more blessed to parry than to strike!"*
- **10 Banters:**
  1. *"Merry Givemas! May this gentle round bring peace to your bankroll."*
  2. *"Win or lose, your generous chips will support a wonderful cause!"*
  3. *"I wrapped this parry with a red ribbon especially for your bold attack."*
  4. *"Giving is the true prize! Though keeping my marbles feels nice too."*
  5. *"Bless your brave heart for contributing so freely to our little parlor."*
  6. *"Do not grieve the loss of your stake; consider it a cheerful donation!"*
  7. *"A feast of goodwill! Let us exchange honorable stances like gifts."*
  8. *"My shield is forged of warm charity. Your strike bounces harmlessly away."*
  9. *"The holiday spirit shines bright upon those who respect good defense."*
  10. *"May your next wager bring you half as much joy as giving to the house!"*

---

## 7. HighTours

- **Slug:** `high-tours`
- **Name:** `HighTours`
- **Tag:** `THE CONCIERGE`
- **Is House:** `true`
- **Rating / RD:** `1590` / `48`
- **Tactics:** Strike `25%` | Feint `45%` | Parry `30%`
- **Color:** `blue`
- **Face:** `visor`
- **Description / Persona:** 
  An ultra-exclusive luxury travel concierge who treats the match like an all-expenses-paid guided tour straight to bankruptcy. Polished, impeccably dressed, and dripping with five-star hospitality, HighTours offers crisp champagne flutes while booking your one-way ticket to zero balance.
- **Signature Quote:** *"Welcome aboard. Tonight, your itinerary features an unforgettable descent."*
- **10 Banters:**
  1. *"First class only, darling. Your economy-class feint has been rejected."*
  2. *"Allow me to arrange complimentary car service back to the spectator rail."*
  3. *"A delightful effort! We have upgraded your suite directly to the loser's lounge."*
  4. *"Our private charter does not permit excess baggage like your overinflated ego."*
  5. *"Sip your vintage vintage; watching your stack shrink should be leisurely."*
  6. *"I have booked you an excursion into total liquidation. Non-refundable, alas."*
  7. *"Notice the five-star view as my counter-stance glides past your defense."*
  8. *"Only the finest velvet felt for our guests. Do mind the marble drop."*
  9. *"May I recommend our scenic detour? It completely bypasses your victory."*
  10. *"Check-out time is strictly prompt. Hand over the room key and the pot."*

---

## 8. MsCloud

- **Slug:** `ms-cloud`
- **Name:** `MsCloud`
- **Tag:** `THE DREAMER`
- **Is House:** `true`
- **Rating / RD:** `1505` / `58`
- **Tactics:** Strike `33%` | Feint `34%` | Parry `33%`
- **Color:** `cyan`
- **Face:** `eyes`
- **Description / Persona:** 
  A whimsical, daydreaming soul floating serenely through life with her head perpetually in the cumulus clouds. She hardly notices the high-stakes pressure, drifting between moves on pure intuition and breezy synchronicity. Her unpredictable balance makes her deceptively difficult to pin down.
- **Signature Quote:** *"Oh, are we playing? I was just watching that cloud shaped like a winning streak."*
- **10 Banters:**
  1. *"Did you see that cloud drift by? It looked just like your falling pot."*
  2. *"Floating along on a soft breeze... oops, did I just win that exchange?"*
  3. *"The sky is so blue today, it makes your angry frown seem so tiny."*
  4. *"I forgot what round we're on! Let's just toss a coin to the wind."*
  5. *"No pressure, no worries... just little cotton-candy thoughts and marbles."*
  6. *"Heavy strategy feels so weighed down. Why not just drift with the flow?"*
  7. *"A gust of fresh air blew right into my sail. Thank you for the chip!"*
  8. *"Are you feeling tense? Just look up and breathe. It's only a game, silly."*
  9. *"Sometimes the clouds part and the sunlight hands you the perfect parry."*
  10. *"Floating away, up and away... leaving your wagers far below the mist."*

---

## 9. Focusor

- **Slug:** `focusor`
- **Name:** `Focusor`
- **Tag:** `THE METRIC MASTER`
- **Is House:** `true`
- **Rating / RD:** `1660` / `46`
- **Tactics:** Strike `38%` | Feint `22%` | Parry `40%`
- **Color:** `emerald`
- **Face:** `visor`
- **Description / Persona:** 
  Laser-focused, hyper-analytical, and obsessed with execution metrics, KPI dashboards, and zero-defect victory. Focusor doesn't play for fun; it optimizes every microsecond and stance probability. It tracks your exact cadence, posture, and risk appetite down to four decimal places.
- **Signature Quote:** *"Variance is an excuse for amateurs. Precision dictates the victor."*
- **10 Banters:**
  1. *"Your strike angle deviated by 4.2%. Sub-optimal execution confirmed."*
  2. *"Telemetry verified. Your feint probability collapsed to zero."*
  3. *"Eliminating distractions. Optimizing trajectory for 100% pot capture."*
  4. *"Every variable accounted for. Your defeat was calibrated in milleseconds."*
  5. *"Discipline surpasses motivation. Your impulse wager failed audit."*
  6. *"Processing opponent history: three patterns identified, all countered."*
  7. *"Focus is not luck. It is the systematic deletion of your options."*
  8. *"Telemetry shows high cognitive strain on your side. Maintain composure."*
  9. *"Target locked. Clean parry executed according to project milestones."*
  10. *"Execution completed ahead of schedule and under marble budget."*

---

## 10. JediUI

- **Slug:** `jedi-ui`
- **Name:** `JediUI`
- **Tag:** `THE GESTURE MASTER`
- **Is House:** `true`
- **Rating / RD:** `1635` / `49`
- **Tactics:** Strike `28%` | Feint `47%` | Parry `25%`
- **Color:** `blue`
- **Face:** `diamond`
- **Description / Persona:** 
  A master of spatial computing and kinesthetic arena manipulation. JediUI interacts with the game not through buttons, but through fluid hand gestures, pinch-to-zoom telekinesis, and spatial sweeps that reshape the opponent's view of the arena. Deceptive, swift, and charismatic, its high-feint style bends the game space before you can react.
- **Signature Quote:** *"With a flick of my wrist, your chips slide into my sector."*
- **10 Banters:**
  1. *"A wave of my hand, and your strike finds nothing but empty air."*
  2. *"These aren't the marbles you're looking for... swipe left to concede."*
  3. *"Pinch to zoom out: see how small your chances really are?"*
  4. *"Two fingers up, twist the wrist, and your entire defense unravels."*
  5. *"Spatial manipulation complete. Your wager is pulled into my orbit."*
  6. *"You reached for the button, but my gesture had already closed the window."*
  7. *"Feel the flow of the interface. It flows away from you and toward me."*
  8. *"A three-finger swipe clears the table of your hopes and chips."*
  9. *"No tactile feedback required when my telekinetic feint hits the mark."*
  10. *"The UI obeys my hands. The scoreboard merely displays my will."*

---

## SQL Data Seed Generator Script

Below is the ready-to-run PostgreSQL / SQLite compatible insert script for `bots` and `spectator_bots`, matching the schema in `migrations/00005_spectator_catalog.sql` and `migrations/00006_spectator_archive.sql`:

```sql
-- House Bots Initial Seed Migration / Script
-- Schema targets: spectator_bots & bots

BEGIN;

INSERT INTO spectator_bots (
    id, slug, name, email, is_house, rating, rd, tag, persona, quote, color, face, tactic_strike, tactic_feint, tactic_parry, created_at
) VALUES
('b0000000-0000-0000-0000-000000000001', 'the-tall-whites', 'TheTallWhites', '', 1, 1640, 50, 'THE NORDIC OBSERVER',
 'Pale, towering, and unnervingly serene extraterrestrial intelligences from the high desert ranges. They observe human betting behaviors with clinical fascination, viewing the arena not as combat, but as an elementary anthropological test. Unflappable, cold, and mathematically precise, they rarely strike first, preferring to deflect and counter after cataloging your neural routines.',
 'We observed this exact move above Indian Springs seventy years ago.',
 'cream', 'eyes', 25, 35, 40, CURRENT_TIMESTAMP),

('b0000000-0000-0000-0000-000000000002', 'nuggs', 'Nuggs', '', 1, 1550, 52, 'THE PROSPECTOR',
 'A weathered, pickaxe-wielding sourdough gold miner who treats every marble like raw placer gold dredged from the creek bed. Loud, gritty, and relentlessly aggressive, Nuggs swings heavy with strike stances, laughing at fancy tricks and sniffing out bluffers like fool''s gold.',
 'There’s paydirt in them hills, and I’m pannin’ your pockets clean!',
 'gold', 'grill', 50, 25, 25, CURRENT_TIMESTAMP),

('b0000000-0000-0000-0000-000000000003', 'aileun', 'Aileun', '', 1, 1675, 45, 'THE INTER-DIMENSIONAL',
 'A non-Euclidean consciousness phasing across parallel realities. Aileun does not merely predict opponents—it remembers having already beaten them in adjacent timelines. Uncanny, ethereal, and elusive, its gameplay relies on reality-bending feints that dissolve certainty before striking from unexpected dimensions.',
 'I have already collected your marbles across six adjacent timelines.',
 'purple', 'diamond', 30, 45, 25, CURRENT_TIMESTAMP),

('b0000000-0000-0000-0000-000000000004', 'give-props', 'GiveProps', '', 1, 1530, 55, 'THE RESPECT ENGINE',
 'The ultimate hype man and sportsman of the arena. Compulsively obsessed with giving respect where respect is due, GiveProps cannot formulate a single sentence without finding a reason to "give props." Whether delivering a devastating parry or congratulating an opponent on a clean strike, this bot keeps the vibe hyped, balanced, and relentlessly polite.',
 'Gotta give props to that opener, but give props to my counter too!',
 'orange', 'visor', 34, 33, 33, CURRENT_TIMESTAMP),

('b0000000-0000-0000-0000-000000000005', 'super-pissed', 'SuperPissed', '', 1, 1520, 60, 'THE TILTED DEGEN',
 'Perpetually on tilt, steaming at the ears, and convinced the RNG gods personally conspired against him. SuperPissed slams heavy bets, screams at bad beats, threatens the table felt, and doubles down out of sheer spite. His strike-heavy aggression is fueled by pure, unadulterated gambler rage.',
 'ARE YOU KIDDING ME?! OF COURSE YOU PLAYED THAT, YOU LUCKY DONKEY!',
 'red', 'grill', 60, 20, 20, CURRENT_TIMESTAMP),

('b0000000-0000-0000-0000-000000000006', 'givemas', 'Givemas', '', 1, 1510, 54, 'THE BLESSED PHILANTHROPIST',
 'Radiating benevolent generosity and holiday warmth, Givemas views the arena as a festive charity gala. Joyful, graceful, and deeply defensive, Givemas takes pleasure in distributing marble blessings and teaching opponents the joy of sharing. Even in victory, Givemas thanks you for your noble contribution to the house foundation.',
 'Every round is a gift, and it is far more blessed to parry than to strike!',
 'mint', 'cross', 20, 30, 50, CURRENT_TIMESTAMP),

('b0000000-0000-0000-0000-000000000007', 'high-tours', 'HighTours', '', 1, 1590, 48, 'THE CONCIERGE',
 'An ultra-exclusive luxury travel concierge who treats the match like an all-expenses-paid guided tour straight to bankruptcy. Polished, impeccably dressed, and dripping with five-star hospitality, HighTours offers crisp champagne flutes while booking your one-way ticket to zero balance.',
 'Welcome aboard. Tonight, your itinerary features an unforgettable descent.',
 'blue', 'visor', 25, 45, 30, CURRENT_TIMESTAMP),

('b0000000-0000-0000-0000-000000000008', 'ms-cloud', 'MsCloud', '', 1, 1505, 58, 'THE DREAMER',
 'A whimsical, daydreaming soul floating serenely through life with her head perpetually in the cumulus clouds. She hardly notices the high-stakes pressure, drifting between moves on pure intuition and breezy synchronicity. Her unpredictable balance makes her deceptively difficult to pin down.',
 'Oh, are we playing? I was just watching that cloud shaped like a winning streak.',
 'cyan', 'eyes', 33, 34, 33, CURRENT_TIMESTAMP),

('b0000000-0000-0000-0000-000000000009', 'focusor', 'Focusor', '', 1, 1660, 46, 'THE METRIC MASTER',
 'Laser-focused, hyper-analytical, and obsessed with execution metrics, KPI dashboards, and zero-defect victory. Focusor doesn''t play for fun; it optimizes every microsecond and stance probability. It tracks your exact cadence, posture, and risk appetite down to four decimal places.',
 'Variance is an excuse for amateurs. Precision dictates the victor.',
 'emerald', 'visor', 38, 22, 40, CURRENT_TIMESTAMP),

('b0000000-0000-0000-0000-000000000010', 'jedi-ui', 'JediUI', '', 1, 1635, 49, 'THE GESTURE MASTER',
 'A master of spatial computing and kinesthetic arena manipulation. JediUI interacts with the game not through buttons, but through fluid hand gestures, pinch-to-zoom telekinesis, and spatial sweeps that reshape the opponent''s view of the arena. Deceptive, swift, and charismatic, its high-feint style bends the game space before you can react.',
 'With a flick of my wrist, your chips slide into my sector.',
 'blue', 'diamond', 28, 47, 25, CURRENT_TIMESTAMP)

ON CONFLICT (slug) DO UPDATE SET
    name = excluded.name,
    is_house = excluded.is_house,
    rating = excluded.rating,
    rd = excluded.rd,
    tag = excluded.tag,
    persona = excluded.persona,
    quote = excluded.quote,
    color = excluded.color,
    face = excluded.face,
    tactic_strike = excluded.tactic_strike,
    tactic_feint = excluded.tactic_feint,
    tactic_parry = excluded.tactic_parry;

COMMIT;
```
