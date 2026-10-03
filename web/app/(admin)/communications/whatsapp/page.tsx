import { ToWorkspace } from '@/components/layout/ToWorkspace';

// WhatsApp is deferred; SMS is the channel in use.
export default function Page() {
  return <ToWorkspace module="communications" section="messages" />;
}
