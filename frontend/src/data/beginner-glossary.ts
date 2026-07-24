export const beginnerGlossary = {
  Evidence: 'A stored news or research event that Jax can inspect.',
  Genuine: 'Collected from a real external source.',
  'Synthetic test': 'Created for controlled testing. It is not live news.',
  Candidate: 'A structured trade idea that still requires human judgement.',
  'Human decision': 'An explicit persisted choice recorded by a person.',
  'Paper plan': 'A hypothetical trade plan. It is not an order or fill.',
  'Hypothetical entry': 'The paper-plan price assumption. It is not a fill.',
  Stop: 'A hypothetical level used to describe planned risk. No position is implied.',
  Target: 'A hypothetical level used to describe planned reward. No position is implied.',
  Quantity: 'The persisted number of hypothetical units in a paper plan.',
  Notional: 'The persisted hypothetical entry value multiplied by quantity.',
  'Planned risk': 'The persisted paper-plan loss assumption, not money lost.',
  'Planned reward': 'The persisted paper-plan gain assumption, not money earned.',
  'Reward/risk': 'The persisted ratio between planned reward and planned risk.',
  Leverage: 'The persisted account exposure assumption. It does not enable execution.',
  'Account-equity assumption': 'A persisted sizing input, not a live account balance.',
  Checkpoint: 'A scheduled retrospective observation after the paper plan began tracking.',
  'Hypothetical return':
    'The calculated percentage movement from the paper-plan entry assumption. It is not realised return.',
  'Hypothetical P&L':
    'The calculated gain or loss using the paper-plan quantity. No money changed hands.',
  MFE: 'The best hypothetical price movement observed during the checkpoint.',
  MAE: 'The worst hypothetical price movement observed during the checkpoint.',
  'Stop touched': 'Recorded market data reached the hypothetical stop level.',
  'Target touched': 'Recorded market data reached the hypothetical target level.',
  'Same-candle ambiguity':
    'The same market-data candle touched both levels, so the exact order cannot be known.',
  Expired:
    'The candidate can no longer receive a new decision; existing retrospective evidence remains valid.',
  'Selected journey': 'Records linked specifically to the candidate being reviewed.',
  'Historical records':
    'Database records from other journeys that are not attributed to this candidate.',
  'Hypothetical outcome':
    'A retrospective calculation showing what could have happened. It is not realised profit or loss.',
  Deduplicated: 'Jax had already stored this event, so it was not stored twice.',
  'Deterministic analysis':
    'Rules or configured logic examined the evidence. No AI model call is implied.',
  'AI analysed': 'Persisted provider or model metadata proves an AI model call occurred.',
  'Unknown assets': 'Jax has no truthful persisted asset mapping for this evidence.',
  'Research only': 'Jax retained the evidence without creating a candidate. This is valid.',
  Rejected: 'Jax did not accept the evidence for further processing.',
  'Duplicate ignored': 'Jax recognised evidence it had already received.',
  Confidence: 'The persisted strength of Jax’s evidence classification, not a trade promise.',
  Normalised: 'Converted into Jax’s consistent research-event shape.',
  Provenance: 'The persisted record of where evidence came from and how it arrived.',
  'Runtime mode': 'Whether Jax is operating in paper or live mode.',
  Execution: 'Whether the system is allowed to create execution-side activity.',
  'Execution worker':
    'The background worker that could process execution instructions when enabled.',
} as const;

export type BeginnerGlossaryTerm = keyof typeof beginnerGlossary;
