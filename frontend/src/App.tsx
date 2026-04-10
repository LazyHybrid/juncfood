import { useEffect, useMemo, useState, type FormEvent } from 'react'
import './App.css'

type Challenge = {
  id: string
  title: string
  description: string
  durationSeconds: number
  setup: string[]
  winCondition: string
  promptLabel?: string
  prompt?: string
}

type Matchup = {
  id: string
  playerOne: string
  playerTwo: string
}

type Vote = {
  voter: string
  choice: string
}

type MatchupResult = {
  matchupId: string
  votes: Vote[]
}

type RoundResponse = {
  challenge: Challenge
  matchups: Matchup[]
  seedOrder: string[]
  instructions: string[]
}

type Ranking = {
  player: string
  score: number
  seed: number
}

type Assignment = {
  player: string
  chore: string
  rank: number
}

type AssignmentResponse = {
  rankings: Ranking[]
  assignments: Assignment[]
  unassignedPlayers: string[]
  unusedChores: string[]
  summary: string
}

const defaultPlayers = ['Alex', 'Sam', 'Jordan', 'Riley'].join('\n')
const defaultChores = ['Choose the playlist', 'Dishes', 'Trash run', 'Bathroom wipe-down'].join('\n')

function parseList(value: string) {
  return value
    .split('\n')
    .map((entry) => entry.trim())
    .filter(Boolean)
}

function isMatchupComplete(
  matchupId: string,
  voters: string[],
  selectedVotes: Record<string, Record<string, string>>,
) {
  return voters.every((voter) => Boolean(selectedVotes[matchupId]?.[voter]))
}

function App() {
  const [playersInput, setPlayersInput] = useState(defaultPlayers)
  const [choresInput, setChoresInput] = useState(defaultChores)
  const [round, setRound] = useState<RoundResponse | null>(null)
  const [assignments, setAssignments] = useState<AssignmentResponse | null>(null)
  const [challengeDeck, setChallengeDeck] = useState<Challenge[]>([])
  const [enabledChallengeIds, setEnabledChallengeIds] = useState<string[]>([])
  const [selectedVotes, setSelectedVotes] = useState<Record<string, Record<string, string>>>({})
  const [errorMessage, setErrorMessage] = useState('')
  const [isLoadingRound, setIsLoadingRound] = useState(false)
  const [isLoadingAssignments, setIsLoadingAssignments] = useState(false)

  const players = useMemo(() => parseList(playersInput), [playersInput])
  const chores = useMemo(() => parseList(choresInput), [choresInput])
  const enabledChallengeCount = enabledChallengeIds.length

  useEffect(() => {
    const loadChallenges = async () => {
      try {
        const response = await fetch('/api/challenges')
        if (!response.ok) {
          throw new Error('Unable to load challenge deck.')
        }

        const payload = (await response.json()) as Challenge[]
        setChallengeDeck(payload)
        setEnabledChallengeIds(payload.map((challenge) => challenge.id))
      } catch {
        setChallengeDeck([])
        setEnabledChallengeIds([])
      }
    }

    void loadChallenges()
  }, [])

  const sidelinedPlayer = useMemo(() => {
    if (!round || round.seedOrder.length % 2 === 0) {
      return ''
    }

    const pairedPlayers = new Set(round.matchups.flatMap((matchup) => [matchup.playerOne, matchup.playerTwo]))
    return round.seedOrder.find((player) => !pairedPlayers.has(player)) ?? ''
  }, [round])

  const allMatchupsPicked = round
    ? round.matchups.every((matchup) =>
        isMatchupComplete(matchup.id, round.seedOrder, selectedVotes),
      )
    : false

  const currentMatchupIndex = useMemo(() => {
    if (!round) {
      return -1
    }

    return round.matchups.findIndex((matchup) => !isMatchupComplete(matchup.id, round.seedOrder, selectedVotes))
  }, [round, selectedVotes])

  const currentMatchup = round && currentMatchupIndex >= 0 ? round.matchups[currentMatchupIndex] : null
  const completedMatchups = round
    ? round.matchups.filter((matchup) => isMatchupComplete(matchup.id, round.seedOrder, selectedVotes)).length
    : 0

  const generateRound = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setIsLoadingRound(true)
    setErrorMessage('')
    setAssignments(null)

    try {
      const response = await fetch('/api/rounds', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          players,
          chores,
          enabledChallengeIds,
        }),
      })

      const payload = (await response.json()) as RoundResponse | { error: string }
      if (!response.ok || 'error' in payload) {
        throw new Error('error' in payload ? payload.error : 'Unable to generate a round.')
      }

      setRound(payload)
      setSelectedVotes({})
    } catch (error) {
      setRound(null)
      setErrorMessage(error instanceof Error ? error.message : 'Unable to generate a round.')
    } finally {
      setIsLoadingRound(false)
    }
  }

  const buildMatchupResults = (currentRound: RoundResponse): MatchupResult[] =>
    currentRound.matchups.map((matchup) => ({
      matchupId: matchup.id,
      votes: currentRound.seedOrder.map((voter) => ({
        voter,
        choice: selectedVotes[matchup.id]?.[voter] ?? '',
      })),
    }))

  const toggleChallenge = (challengeId: string) => {
    setEnabledChallengeIds((current) =>
      current.includes(challengeId)
        ? current.filter((id) => id !== challengeId)
        : [...current, challengeId],
    )
  }

  const assignChores = async () => {
    if (!round) {
      return
    }

    setIsLoadingAssignments(true)
    setErrorMessage('')

    try {
      const response = await fetch('/api/assignments', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          players,
          chores,
          seedOrder: round.seedOrder,
          results: buildMatchupResults(round),
        }),
      })

      const payload = (await response.json()) as AssignmentResponse | { error: string }
      if (!response.ok || 'error' in payload) {
        throw new Error('error' in payload ? payload.error : 'Unable to assign chores.')
      }

      setAssignments(payload)
    } catch (error) {
      setAssignments(null)
      setErrorMessage(error instanceof Error ? error.message : 'Unable to assign chores.')
    } finally {
      setIsLoadingAssignments(false)
    }
  }

  return (
    <main className="app-shell">
      <section className="hero-panel">
        <div className="hero-copy">
          <p className="eyebrow">Hackathon game night</p>
          <h1>Settle chores with ridiculous mini-challenges.</h1>
          <p className="hero-text">
            Build a roster, rank the chores from best to worst, and let a random challenge decide who gets first pick.
          </p>
        </div>

        <div className="hero-stats">
          <div>
            <span className="stat-label">Players loaded</span>
            <strong>{players.length}</strong>
          </div>
          <div>
            <span className="stat-label">Chores queued</span>
            <strong>{chores.length}</strong>
          </div>
          <div>
            <span className="stat-label">Challenge deck</span>
            <strong>{enabledChallengeCount}</strong>
          </div>
        </div>
      </section>

      <section className="grid-layout">
        <form className="panel form-panel" onSubmit={generateRound}>
          <div className="panel-heading">
            <div>
              <p className="eyebrow">Setup</p>
              <h2>Tonight&apos;s lineup</h2>
            </div>
            <button className="primary-button" type="submit" disabled={isLoadingRound}>
              {isLoadingRound ? 'Drawing...' : 'Draw challenge round'}
            </button>
          </div>

          <label>
            Players
            <textarea
              value={playersInput}
              onChange={(event) => setPlayersInput(event.target.value)}
              rows={7}
              placeholder="One player per line"
            />
          </label>

          <label>
            Chores in award order
            <textarea
              value={choresInput}
              onChange={(event) => setChoresInput(event.target.value)}
              rows={7}
              placeholder="Top line = best outcome"
            />
          </label>

          <p className="helper-text">
            Enter chores in the order they should be awarded. Players with more votes take the earlier slots.
          </p>
          <p className="helper-text">Enabled challenges: {enabledChallengeCount} of {challengeDeck.length || 0}</p>

          {errorMessage ? <p className="error-banner">{errorMessage}</p> : null}
        </form>

        <aside className="panel deck-panel">
          <div className="panel-heading compact">
            <div>
              <p className="eyebrow">Randomizer</p>
              <h2>Challenge deck</h2>
            </div>
          </div>

          <div className="challenge-tags">
            {challengeDeck.map((challenge) => (
              <button
                key={challenge.id}
                type="button"
                className={enabledChallengeIds.includes(challenge.id) ? 'tag challenge-toggle is-enabled' : 'tag challenge-toggle is-disabled'}
                onClick={() => toggleChallenge(challenge.id)}
                aria-pressed={enabledChallengeIds.includes(challenge.id)}
              >
                {challenge.title}
              </button>
            ))}
            {challengeDeck.length === 0 ? <span className="tag muted">Backend offline. Starter deck still works once the API is up.</span> : null}
          </div>
          <p className="helper-text">Tap a challenge to toggle it on or off. Disabled challenges will not be drawn.</p>
        </aside>
      </section>

      {round ? (
        <section className="grid-layout results-layout">
          <article className="panel challenge-panel">
            <div className="panel-heading compact">
              <div>
                <p className="eyebrow">Round card</p>
                <h2>{round.challenge.title}</h2>
              </div>
              <span className="time-pill">{round.challenge.durationSeconds}s</span>
            </div>

            <p className="challenge-description">{round.challenge.description}</p>
            {round.challenge.prompt ? (
              <p className="scenario-callout">{round.challenge.promptLabel ?? 'Prompt'}: {round.challenge.prompt}</p>
            ) : null}
            <p className="win-condition">{round.challenge.winCondition}</p>

            <div className="detail-block">
              <h3>Setup</h3>
              <ul>
                {round.challenge.setup.map((item) => (
                  <li key={item}>{item}</li>
                ))}
              </ul>
            </div>

            <div className="detail-block">
              <h3>How this round works</h3>
              <ul>
                {round.instructions.map((instruction) => (
                  <li key={instruction}>{instruction}</li>
                ))}
              </ul>
            </div>

            <div className="seed-strip">
              {round.seedOrder.map((player, index) => (
                <span key={player} className="seed-pill">
                  {index + 1}. {player}
                </span>
              ))}
            </div>
          </article>

          <article className="panel matchup-panel">
            <div className="panel-heading">
              <div>
                <p className="eyebrow">Results</p>
                <h2>Cast votes for each matchup</h2>
                {round ? (
                  <p className="helper-text">
                    {completedMatchups} of {round.matchups.length} matchups scored
                  </p>
                ) : null}
              </div>
              <button
                className="primary-button"
                type="button"
                onClick={assignChores}
                disabled={!allMatchupsPicked || isLoadingAssignments}
              >
                {isLoadingAssignments ? 'Assigning...' : 'Lock chore assignments'}
              </button>
            </div>

            <div className="matchup-list">
              {currentMatchup ? (
                <section key={currentMatchup.id} className="matchup-card">
                  <p className="matchup-label">
                    {currentMatchup.id.replace('-', ' ')} · matchup {currentMatchupIndex + 1} of {round.matchups.length}
                  </p>
                  <div className="vote-grid">
                    {round.seedOrder.map((voter) => (
                      <div key={`${currentMatchup.id}-${voter}`} className="vote-row">
                        <span className="voter-name">{voter}</span>
                        <div className="duel-buttons">
                          {[currentMatchup.playerOne, currentMatchup.playerTwo].map((player) => (
                            <button
                              key={player}
                              type="button"
                              className={selectedVotes[currentMatchup.id]?.[voter] === player ? 'duel-button selected' : 'duel-button'}
                              onClick={() =>
                                setSelectedVotes((current) => ({
                                  ...current,
                                  [currentMatchup.id]: {
                                    ...(current[currentMatchup.id] ?? {}),
                                    [voter]: player,
                                  },
                                }))
                              }
                            >
                              {player}
                            </button>
                          ))}
                        </div>
                      </div>
                    ))}
                  </div>
                </section>
              ) : (
                <section className="matchup-card matchup-card-complete">
                  <p className="matchup-label">All matchups are scored</p>
                  <p className="helper-text">Lock chore assignments when you are ready to rank the players.</p>
                </section>
              )}
            </div>

            {round.matchups.length > 1 ? (
              <div className="matchup-progress-strip">
                {round.matchups.map((matchup, index) => (
                  <span
                    key={matchup.id}
                    className={isMatchupComplete(matchup.id, round.seedOrder, selectedVotes) ? 'progress-pill complete' : 'progress-pill'}
                  >
                    {index + 1}
                  </span>
                ))}
              </div>
            ) : null}

            <p className="helper-text">Every player votes on every matchup. Total votes decide the ranking, and ties are broken randomly.</p>
            {sidelinedPlayer ? <p className="helper-text">{sidelinedPlayer} sits out the challenge itself, but still gets a vote.</p> : null}
          </article>
        </section>
      ) : null}

      {assignments ? (
        <section className="grid-layout final-layout">
          <article className="panel">
            <div className="panel-heading compact">
              <div>
                <p className="eyebrow">Outcome</p>
                <h2>Who does what</h2>
              </div>
            </div>

            <div className="assignment-list">
              {assignments.assignments.map((entry) => (
                <div key={`${entry.player}-${entry.chore}`} className="assignment-row">
                  <span className="rank-badge">#{entry.rank}</span>
                  <div>
                    <strong>{entry.player}</strong>
                    <p>{entry.chore}</p>
                  </div>
                </div>
              ))}
            </div>

            <p className="helper-text">{assignments.summary}</p>

            {assignments.unassignedPlayers.length > 0 ? (
              <p className="helper-text">No chore assigned: {assignments.unassignedPlayers.join(', ')}</p>
            ) : null}

            {assignments.unusedChores.length > 0 ? (
              <p className="helper-text">Saved for later: {assignments.unusedChores.join(', ')}</p>
            ) : null}
          </article>

          <article className="panel ranking-panel">
            <div className="panel-heading compact">
              <div>
                <p className="eyebrow">Scoreboard</p>
                <h2>Ranked finish</h2>
              </div>
            </div>

            <ol className="ranking-list">
              {assignments.rankings.map((entry) => (
                <li key={entry.player}>
                  <span>{entry.player}</span>
                  <span>{entry.score} votes</span>
                </li>
              ))}
            </ol>
          </article>
        </section>
      ) : null}
    </main>
  )
}

export default App
