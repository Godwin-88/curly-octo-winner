import { ToWorkspace } from '@/components/layout/ToWorkspace';

// The WhatsApp inbox is deferred with WhatsApp.
export default function Page() {
  return <ToWorkspace module="communications" section="messages" />;
}
