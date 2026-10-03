import { ToWorkspace } from '@/components/layout/ToWorkspace';

// The old SMS campaign builder: now the New SMS form.
export default function Page() {
  return <ToWorkspace module="communications" section="messages" action="create" />;
}
