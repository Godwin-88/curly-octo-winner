'use client';

import { useParams } from 'next/navigation';
import { ToWorkspace } from '@/components/layout/ToWorkspace';

// Old address of one message: forwards to it in the workspace.
export default function Page() {
  const { id } = useParams<{ id: string }>();
  return <ToWorkspace module="communications" section="messages" record={id} />;
}
