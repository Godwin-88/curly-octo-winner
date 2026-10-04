import { ToWorkspace } from '@/components/layout/ToWorkspace';

// Finance lives in the list → view → edit workspace; this address forwards there.
export default function Page() {
  return <ToWorkspace module="finance" section="payments" />;
}
