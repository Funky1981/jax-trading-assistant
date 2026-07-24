export const beginnerGlossary = {
  Evidence: 'A stored news or research event that Jax can inspect.',
  Genuine: 'Collected from a real external source.',
  'Synthetic test': 'Created for controlled testing. It is not live news.',
  Candidate: 'A structured trade idea that still requires human judgement.',
  'Paper plan': 'A hypothetical trade plan. It is not an order or fill.',
  'Hypothetical outcome':
    'A retrospective calculation showing what could have happened. It is not realised profit or loss.',
  Deduplicated: 'Jax had already stored this event, so it was not stored twice.',
  'Runtime mode': 'Whether Jax is operating in paper or live mode.',
  Execution: 'Whether the system is allowed to create execution-side activity.',
  'Execution worker':
    'The background worker that could process execution instructions when enabled.',
} as const;

export type BeginnerGlossaryTerm = keyof typeof beginnerGlossary;
