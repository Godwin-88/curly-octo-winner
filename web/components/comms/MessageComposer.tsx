'use client';

interface Props {
  content: string;
  setContent: (content: string) => void;
  scheduledAt?: string;
  setScheduledAt: (date?: string) => void;
}

const VARIABLES = [
  { label: 'Parent Name', value: '{{parent_name}}' },
  { label: 'Learner Name', value: '{{learner_name}}' },
  { label: 'Fee Balance', value: '{{fee_balance}}' },
  { label: 'Class', value: '{{class}}' },
];

export default function MessageComposer({
  content,
  setContent,
  scheduledAt,
  setScheduledAt,
}: Props) {
  const charCount = content.length;
  const smsUnits = charCount === 0 ? 0 : Math.min(Math.ceil(charCount / 160), 3);

  const insertVariable = (variable: string) => {
    setContent(content + variable);
  };

  return (
    <div className="card p-6 max-w-2xl">
      <h2 className="text-lg font-semibold mb-4">Compose Message</h2>

      <div className="space-y-4">
        <div>
          <label className="label">Message Content</label>
          <textarea
            className="input min-h-[150px]"
            value={content}
            onChange={(e) => setContent(e.target.value)}
            placeholder="Type your message here..."
            maxLength={480}
          />
          <div className="flex justify-between mt-1 text-sm">
            <span className="text-gray-500">
              {charCount} / 480 characters
            </span>
            <span className="text-gray-500">
              {smsUnits} SMS unit{smsUnits !== 1 ? 's' : ''}
            </span>
          </div>
        </div>

        <div>
          <label className="label">Insert Variables</label>
          <div className="flex flex-wrap gap-2">
            {VARIABLES.map((v) => (
              <button
                key={v.value}
                className="px-3 py-1 bg-blue-50 text-blue-700 rounded-md text-sm hover:bg-blue-100"
                onClick={() => insertVariable(v.value)}
              >
                {v.label}
              </button>
            ))}
          </div>
        </div>

        <div>
          <label className="label">Schedule</label>
          <div className="flex items-center gap-3">
            <input
              type="checkbox"
              checked={!!scheduledAt}
              onChange={(e) => {
                if (!e.target.checked) setScheduledAt(undefined);
              }}
              className="w-4 h-4"
            />
            <span className="text-sm text-gray-600">Schedule for later</span>
          </div>
          {scheduledAt && (
            <input
              type="datetime-local"
              className="input mt-2"
              value={scheduledAt}
              onChange={(e) => setScheduledAt(e.target.value)}
            />
          )}
        </div>
      </div>
    </div>
  );
}