<script lang="ts">
  // Plus-alias so the address is filterable / disposable if it attracts spam.
  // The repo is public, so this address is visible in source regardless --
  // superseded later by a server-side Resend feedback form (see CLAUDE.md).
  const contactEmail = 'nakamajamey+flockdeck@gmail.com'
  const mailto = `mailto:${contactEmail}`
</script>

<div class="about">
  <h1>About FlockDeck</h1>

  <section>
    <h2>What FlockDeck is</h2>
    <p>
      FlockDeck helps you learn to identify birds by ear and by eye. You hear a
      recording or see a photo, name the species, and reveal the answer. Spaced
      repetition tracks what you know and brings each bird back right before
      you'd forget it.
    </p>
    <p class="contact-line">
      Found a bug or have feedback? <a href={mailto}>Get in touch</a>.
    </p>
  </section>

  <section id="getting-started">
    <h2>Getting started</h2>
    <p>
      Practice happens in <strong>decks</strong> — a set of species to drill. The
      fastest way to begin is to clone a <strong>Starter Deck</strong> from the
      home page (regional presets sourced from eBird) and start practicing right
      away.
    </p>
    <p>
      You can also build your own: go to <a href="/decks">Decks</a> to create one,
      then use <a href="/explore">Explore</a> to find and add species — by region
      or by searching the catalog.
    </p>
  </section>

  <section>
    <h2>Audio &amp; image lanes</h2>
    <p>
      Every species can be practiced in two <strong>lanes</strong>: audio (learn
      the song or call by ear) and image (learn the bird by sight). The two lanes
      are scheduled independently — you might know a bird's song cold while still
      working on recognizing its photo. You can toggle which lanes a species uses
      per species.
    </p>
  </section>

  <section>
    <h2>The practice loop</h2>
    <p>
      Each card plays a recording or shows a photo. Type the species into the
      typeahead, then reveal to check yourself and advance to the next card.
      There's no self-grading: a correct guess is rated automatically, and so is
      a miss.
    </p>
  </section>

  <section>
    <h2>How scheduling works</h2>
    <p>
      Under the hood FlockDeck uses <abbr title="Free Spaced Repetition Scheduler"
        >FSRS</abbr
      >, a spaced-repetition algorithm. A correct answer is rated
      <em>Good</em> and pushes the card further into the future; a miss is rated
      <em>Again</em> and brings it back soon. That's why the "due" counts on your
      decks rise and fall over time — the app is spacing each bird to the moment
      you're most likely to have just started forgetting it.
    </p>
  </section>

  <section>
    <h2>Credits</h2>
    <p>FlockDeck is built on freely available birding data:</p>
    <ul>
      <li><strong>eBird</strong> — regional checklists behind the Starter Decks.</li>
      <li>
        <strong>Xeno-canto</strong> — community recordings, used under their
        Creative Commons licenses.
      </li>
      <li><strong>Macaulay Library</strong> — species photos.</li>
    </ul>
    <p class="credit-note">
      Each recording and photo shows its individual contributor's credit in the
      quiz itself; this page is the project-level attribution.
    </p>
    <p>Questions or a bug to report? <a href={mailto}>Get in touch</a>.</p>
  </section>
</div>

<style>
  .about {
    display: flex;
    flex-direction: column;
    gap: 1.5rem;
    line-height: 1.6;
  }
  h1 {
    font-size: 1.25rem;
    font-weight: 700;
    color: var(--text);
    margin: 0;
  }
  h2 {
    font-size: 1.0625rem;
    font-weight: 700;
    color: var(--text);
    margin: 0 0 0.5rem;
  }
  section {
    scroll-margin-top: 1rem;
  }
  p {
    color: var(--text-secondary);
    margin: 0 0 0.5rem;
  }
  ul {
    color: var(--text-secondary);
    margin: 0 0 0.5rem;
    padding-left: 1.25rem;
  }
  li {
    margin-bottom: 0.25rem;
  }
  a {
    color: var(--accent);
  }
  .contact-line,
  .credit-note {
    font-size: 0.875rem;
    color: var(--text-muted);
  }
  abbr {
    text-decoration: underline dotted;
    cursor: help;
  }
</style>
