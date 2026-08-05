import Link from 'next/link';
import { MessageSquare, MessageCircle, Inbox } from 'lucide-react';

export default function CommunicationsPage() {
  return (
    <div>
      <h1 className="text-2xl font-bold mb-6">Communications Hub</h1>
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        <Link href="/communications/sms" className="card p-6 hover:shadow-md transition-shadow">
          <MessageSquare className="w-8 h-8 text-blue-600 mb-3" />
          <h3 className="font-semibold text-lg">SMS Campaigns</h3>
          <p className="text-sm text-gray-500 mt-1">
            Send targeted SMS to parents, schedule campaigns, and track delivery.
          </p>
        </Link>
        <Link href="/communications/whatsapp" className="card p-6 hover:shadow-md transition-shadow">
          <MessageCircle className="w-8 h-8 text-green-600 mb-3" />
          <h3 className="font-semibold text-lg">WhatsApp Broadcast</h3>
          <p className="text-sm text-gray-500 mt-1">
            Send rich media messages via WhatsApp Business API.
          </p>
        </Link>
        <Link href="/communications/inbox" className="card p-6 hover:shadow-md transition-shadow">
          <Inbox className="w-8 h-8 text-purple-600 mb-3" />
          <h3 className="font-semibold text-lg">Conversation Inbox</h3>
          <p className="text-sm text-gray-500 mt-1">
            Manage two-way WhatsApp conversations with parents.
          </p>
        </Link>
      </div>
    </div>
  );
}