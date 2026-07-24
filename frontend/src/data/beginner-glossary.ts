export const beginnerGlossary = {
  Evidence: 'A stored news or research event that Jax can inspect.',
  Genuine: 'Collected from a real external source.',
  'Synthetic test': 'Created for controlled testing. It is not live news.',
  Candidate: 'A structured trade idea that still requires human judgement.',
  'Paper plan': 'A hypothetical trade plan. It is not an order or fill.',
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
