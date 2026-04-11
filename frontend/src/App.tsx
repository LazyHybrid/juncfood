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

type ScoreEntry = {
  voter: string
  player: string
  score: number
}

type RoundResponse = {
  challenge: Challenge
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

type PlayerAssignmentGroup = {
  player: string
  rank: number
  chores: string[]
}

const defaultPlayers = ['Frodo', 'Gandalf', 'Sauron', 'Aragorn', 'Legolas', 'Orc #42'].join('\n')
const defaultChores = ['Vibe Manager', 'Water Wizard', 'Chaos Cleaner', 'Dish Gremlin', 'Floor Goblin', 'Trash Titan'].join('\n')

function parseList(value: string) {
  return value
    .split('\n')
    .map((entry) => entry.trim())
    .filter(Boolean)
}

function isScoringComplete(
  players: string[],
  selectedScores: Record<string, Record<string, number>>,
) {
  return players.every((voter) =>
    players
      .filter((player) => player !== voter)
      .every((player) => Boolean(selectedScores[voter]?.[player])),
  )
}

function isVoterScoringComplete(
  voter: string,
  players: string[],
  selectedScores: Record<string, Record<string, number>>,
) {
  return players
    .filter((player) => player !== voter)
    .every((player) => Boolean(selectedScores[voter]?.[player]))
}

function App() {
  const [playersInput, setPlayersInput] = useState(defaultPlayers)
  const [choresInput, setChoresInput] = useState(defaultChores)
  const [round, setRound] = useState<RoundResponse | null>(null)
  const [assignments, setAssignments] = useState<AssignmentResponse | null>(null)
  const [challengeDeck, setChallengeDeck] = useState<Challenge[]>([])
  const [enabledChallengeIds, setEnabledChallengeIds] = useState<string[]>([])
  const [selectedScores, setSelectedScores] = useState<Record<string, Record<string, number>>>({})
  const [currentVoterIndex, setCurrentVoterIndex] = useState(0)
  const [errorMessage, setErrorMessage] = useState('')
  const [isLoadingRound, setIsLoadingRound] = useState(false)
  const [isLoadingAssignments, setIsLoadingAssignments] = useState(false)

  const players = useMemo(() => parseList(playersInput), [playersInput])
  const chores = useMemo(() => parseList(choresInput), [choresInput])
  const enabledChallengeCount = enabledChallengeIds.length

  const groupedAssignments = useMemo<PlayerAssignmentGroup[]>(() => {
    if (!assignments) {
      return []
    }

    const grouped = new Map<string, PlayerAssignmentGroup>()
    for (const entry of assignments.assignments) {
      const existing = grouped.get(entry.player)
      if (existing) {
        existing.chores.push(entry.chore)
        continue
      }

      grouped.set(entry.player, {
        player: entry.player,
        rank: entry.rank,
        chores: [entry.chore],
      })
    }

    return [...grouped.values()].sort((left, right) => left.rank - right.rank)
  }, [assignments])

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

  const allScoresPicked = round
    ? isScoringComplete(round.seedOrder, selectedScores)
    : false
  const currentVoter = round ? round.seedOrder[currentVoterIndex] : null
  const currentVoterComplete = round && currentVoter
    ? isVoterScoringComplete(currentVoter, round.seedOrder, selectedScores)
    : false

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
      setSelectedScores({})
      setCurrentVoterIndex(0)
    } catch (error) {
      setRound(null)
      setErrorMessage(error instanceof Error ? error.message : 'Unable to generate a round.')
    } finally {
      setIsLoadingRound(false)
    }
  }

  const buildScores = (currentRound: RoundResponse): ScoreEntry[] =>
    currentRound.seedOrder.flatMap((voter) =>
      currentRound.seedOrder
        .filter((player) => player !== voter)
        .map((player) => ({
          voter,
          player,
          score: selectedScores[voter]?.[player] ?? 0,
        })),
    )

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
          scores: buildScores(round),
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
          <p className="eyebrow">Stupid Hack 2026</p>
          <h1>Chore-Ha</h1>
          <p className="hero-text">
            Add the players, draw a challenge from the challenge deck, play the games, and may the best player skip the chores!
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
              placeholder="Top line = best chore after Relax"
            />
          </label>

          <p className="helper-text">
            First place gets Relax. Enter the remaining chores in the order they should be assigned.
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

            <div className="detail-block">
              <h3>Players</h3>
              <div className="seed-strip">
                {round.seedOrder.map((player, index) => (
                  <span key={player} className="seed-pill">
                    {index + 1}. {player}
                  </span>
                ))}
              </div>
            </div>
          </article>

          <article className="panel matchup-panel">
            <div className="panel-heading">
              <div>
                <p className="eyebrow">Results</p>
                <h2>Score one player at a time</h2>
                <p className="helper-text">Pass the device around. Each player scores every other player from 1 to 5.</p>
              </div>
              <button
                className="primary-button"
                type="button"
                onClick={assignChores}
                disabled={!allScoresPicked || isLoadingAssignments}
              >
                {isLoadingAssignments ? 'Assigning...' : 'Lock chore assignments'}
              </button>
            </div>

            <div className="matchup-list">
              {round ? (
                <section className="matchup-card matchup-card-complete">
                  <p className="matchup-label">One round, everybody plays</p>
                  <div className="matchup-progress-strip">
                    {round.seedOrder.map((player, index) => (
                      <button
                        key={player}
                        type="button"
                        className={[
                          'progress-pill',
                          index === currentVoterIndex ? 'current' : '',
                          isVoterScoringComplete(player, round.seedOrder, selectedScores) ? 'complete' : '',
                        ].filter(Boolean).join(' ')}
                        onClick={() => setCurrentVoterIndex(index)}
                      >
                        {index + 1}
                      </button>
                    ))}
                  </div>

                  {currentVoter ? (
                    <div className="vote-grid">
                      <div className="vote-row vote-row-stack">
                        <span className="voter-name">Now scoring: {currentVoter}</span>
                        <p className="helper-text">Player {currentVoterIndex + 1} of {round.seedOrder.length}</p>
                        <div className="score-grid">
                          {round.seedOrder.filter((player) => player !== currentVoter).map((player) => (
                            <div key={`${currentVoter}-${player}`} className="score-entry">
                              <span className="score-target">{player}</span>
                              <div className="duel-buttons score-buttons">
                                {[1, 2, 3, 4, 5].map((score) => (
                                  <button
                                    key={score}
                                    type="button"
                                    className={selectedScores[currentVoter]?.[player] === score ? 'duel-button selected score-button' : 'duel-button score-button'}
                                    onClick={() =>
                                      setSelectedScores((current) => ({
                                        ...current,
                                        [currentVoter]: {
                                          ...(current[currentVoter] ?? {}),
                                          [player]: score,
                                        },
                                      }))
                                    }
                                  >
                                    {score}
                                  </button>
                                ))}
                              </div>
                            </div>
                          ))}
                        </div>

                        <div className="scoring-nav">
                          <button
                            className="duel-button"
                            type="button"
                            onClick={() => setCurrentVoterIndex((current) => Math.max(0, current - 1))}
                            disabled={currentVoterIndex === 0}
                          >
                            Previous player
                          </button>
                          <button
                            className="duel-button"
                            type="button"
                            onClick={() => setCurrentVoterIndex((current) => Math.min(round.seedOrder.length - 1, current + 1))}
                            disabled={currentVoterIndex === round.seedOrder.length - 1}
                          >
                            {currentVoterComplete ? 'Next player' : 'Skip ahead'}
                          </button>
                        </div>
                      </div>
                    </div>
                  ) : null}
                </section>
              ) : (
                <section className="matchup-card matchup-card-complete">
                  <p className="matchup-label">All matchups are scored</p>
                  <p className="helper-text">Lock chore assignments when you are ready to rank the players.</p>
                </section>
              )}
            </div>

            <p className="helper-text">Scores are added together across all voters. Higher total score means a better finish, and ties are broken randomly.</p>
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
              {groupedAssignments.map((entry) => (
                <div key={entry.player} className="assignment-row">
                  <span className="rank-badge">#{entry.rank}</span>
                  <div>
                    <strong>{entry.player}</strong>
                    <p>{entry.chores.join(', ')}</p>
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
                  <span>{entry.score} points</span>
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
