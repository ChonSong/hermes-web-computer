#!/usr/bin/env python3
"""
Fun-Audio-Chat bridge: relays WebSocket audio between Go backend and Fun-Audio-Chat.

Go sends JSON-RPC audio envelopes → this script translates to Fun-Audio-Chat's
native binary protocol (HANDSHAKE/AUDIO/TEXT/CONTROL) on localhost:11235.

Usage:
    python bridge.py --funaudio-ws ws://localhost:11235/api/chat
"""
import asyncio
import json
import logging
import struct
from dataclasses import dataclass
from typing import Optional

import websockets

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger("audio-bridge")

# Fun-Audio-Chat protocol constants
class MessageType:
    HANDSHAKE = 0x00
    AUDIO = 0x01
    TEXT = 0x02
    CONTROL = 0x03
    METADATA = 0x04
    ERROR = 0x05
    PING = 0x06

class ControlMessage:
    START = 0x00
    END_TURN = 0x01
    PAUSE = 0x02
    RESTART = 0x03

@dataclass
class AudioSession:
    id: str
    fa_client: Optional[websockets.WebSocketClientProtocol] = None
    active: bool = False

class AudioBridge:
    def __init__(self, funaudio_url: str = "ws://localhost:11235/api/chat"):
        self.funaudio_url = funaudio_url
        self.sessions: dict[str, AudioSession] = {}

    async def handle_envelope(self, session_id: str, envelope: dict):
        """Handle a JSON-RPC audio envelope from Go backend."""
        method = envelope.get("method", "")

        if method == "audio.stream":
            # Extract Opus chunk from params
            opus_data = envelope.get("params", {}).get("opus_chunk", "")
            await self._send_audio(session_id, opus_data)

        elif method == "audio.interrupt":
            # Send PAUSE control message to abort mid-inference
            await self._send_control(session_id, ControlMessage.PAUSE)

        elif method == "audio.start":
            await self._send_control(session_id, ControlMessage.START)

    async def _send_audio(self, session_id: str, opus_data: str):
        """Send audio chunk to Fun-Audio-Chat via binary protocol."""
        session = self.sessions.get(session_id)
        if not session or not session.fa_client:
            logger.warning(f"No active Fun-Audio-Chat connection for {session_id}")
            return

        # Encode as: MessageType.AUDIO (1 byte) + length (2 bytes) + payload
        payload = bytes([MessageType.AUDIO])
        payload += struct.pack(">H", len(opus_data))
        payload += opus_data.encode("utf-8") if isinstance(opus_data, str) else opus_data

        await session.fa_client.send(payload)

    async def _send_control(self, session_id: str, control_type: int):
        """Send control message to Fun-Audio-Chat."""
        session = self.sessions.get(session_id)
        if not session or not session.fa_client:
            return

        payload = bytes([MessageType.CONTROL, control_type])
        await session.fa_client.send(payload)

    async def connect_funaudio(self, session_id: str) -> websockets.WebSocketClientProtocol:
        """Establish WebSocket connection to Fun-Audio-Chat server."""
        client = await websockets.connect(self.funaudio_url)
        self.sessions[session_id] = AudioSession(id=session_id, fa_client=client, active=True)

        # Read handshake response
        handshake = await client.recv()
        logger.info(f"Fun-Audio-Chat handshake: {handshake[:10]}")
        return client

    async def disconnect_funaudio(self, session_id: str):
        """Close Fun-Audio-Chat connection."""
        session = self.sessions.get(session_id)
        if session and session.fa_client:
            await session.fa_client.close()
            session.active = False

if __name__ == "__main__":
    import argparse
    parser = argparse.ArgumentParser()
    parser.add_argument("--funaudio-ws", default="ws://localhost:11235/api/chat")
    args = parser.parse_args()

    bridge = AudioBridge(funaudio_url=args.funaudio_ws)
    logger.info("Audio bridge ready. Fun-Audio-Chat: %s", args.funaudio_ws)
