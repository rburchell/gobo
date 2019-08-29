package main

import "fmt"

type wireType int

// These match up with the protobuf wiretypes.
// That's also why they are not sequential.
const (
	varInt          wireType = 0
	fixed64         wireType = 1
	lengthDelimited wireType = 2
	fixed32                  = 5
)

func preamble() {
	fmt.Printf(`
#include <cstdint>
#include <iostream>
#include <cstring>
#include <vector>

enum class WireType {
	VarInt = %d,
	Fixed64 = %d,
	LengthDelimited = %d,
	Fixed32 = %d,
};
`, varInt, fixed64, lengthDelimited, fixed32)

	fmt.Printf(`
struct VarInt
{
	uint64_t v;
};

struct Fixed64
{
	uint64_t v;
};

struct Fixed32
{
	uint32_t v;
};
struct LengthDelimited
{
	std::vector<uint8_t> data;
};

class StreamWriter
{
public:
	void write(const VarInt& v);
	void write(const LengthDelimited& v);
	void write(const Fixed64& v);
	void write(const Fixed32& v);
	std::vector<uint8_t> buffer() const { return m_buffer; }
private:
	std::vector<uint8_t> m_buffer;
};

class StreamReader
{
public:
	StreamReader(const std::vector<uint8_t>& buffer);
	void read(VarInt& v);
	void read(LengthDelimited& v);
	void read(Fixed64& v);
	void read(Fixed32& v);
	std::vector<uint8_t> buffer() const;
	bool is_eof() const;
	void start_transaction();
	bool commit_transaction();
private:
	uint8_t readb();
	std::vector<uint8_t> m_buffer;
	size_t m_pos = 0;
	std::vector<size_t> m_pos_stack;
};

inline StreamReader::StreamReader(const std::vector<uint8_t>& buffer)
	: m_buffer(buffer)
{}

inline std::vector<uint8_t> StreamReader::buffer() const
{ return m_buffer; }

inline bool StreamReader::is_eof() const
{ return m_pos >= m_buffer.size(); }

inline void StreamReader::start_transaction()
{ m_pos_stack.push_back(m_pos); }

inline bool StreamReader::commit_transaction()
{
	if (m_pos > m_buffer.size()) {
		m_pos = m_pos_stack.back();
		return false;
	} else {
		m_pos_stack.pop_back();
		return true;
	}
}

inline uint8_t StreamReader::readb()
{
	if (is_eof()) {
		return 0;
	}
	return m_buffer[m_pos++];
}

inline void StreamWriter::write(const VarInt& t)
{
	uint64_t val = t.v;
	int hibit;
	do {
		hibit = 0;
		if (val & ~0x7f)
			hibit = 0x80;
		m_buffer.push_back(uint8_t((val & 0x7f) | hibit));
		val >>= 7;
	} while (hibit);
}

inline void StreamWriter::write(const LengthDelimited& t)
{
	VarInt sz;
	sz.v = t.data.size();
	write(sz);
	m_buffer.insert(m_buffer.end(), t.data.begin(), t.data.end());
}

inline void StreamWriter::write(const Fixed64& t)
{
	uint32_t p1 = static_cast<uint32_t>(t.v);
	uint32_t p2 = static_cast<uint32_t>(t.v >> 32);
	m_buffer.push_back(static_cast<uint8_t>(p1));
	m_buffer.push_back(static_cast<uint8_t>(p1 >> 8));
	m_buffer.push_back(static_cast<uint8_t>(p1 >> 16));
	m_buffer.push_back(static_cast<uint8_t>(p1 >> 24));
	m_buffer.push_back(static_cast<uint8_t>(p2));
	m_buffer.push_back(static_cast<uint8_t>(p2 >> 8));
	m_buffer.push_back(static_cast<uint8_t>(p2 >> 16));
	m_buffer.push_back(static_cast<uint8_t>(p2 >> 24));
}

inline void StreamWriter::write(const Fixed32& t)
{
	m_buffer.push_back(static_cast<uint8_t>(t.v));
	m_buffer.push_back(static_cast<uint8_t>(t.v >> 8));
	m_buffer.push_back(static_cast<uint8_t>(t.v >> 16));
	m_buffer.push_back(static_cast<uint8_t>(t.v >> 24));
}

inline void StreamReader::read(Fixed32& t)
{
	t.v = 0;
	t.v = (static_cast<uint32_t>(readb())) |
		 (static_cast<uint32_t>(readb()) << 8) |
		 (static_cast<uint32_t>(readb()) << 16) |
		 (static_cast<uint32_t>(readb()) << 24);
}

inline void StreamReader::read(VarInt& t)
{
	t.v = 0;
	int shift = 0;

	while (!is_eof()) {
		uint8_t c = readb();
		t.v |= (uint64_t)(c & 0x7f) << shift;
		if ((c & 0x80) == 0)
			break;
		shift += 7;
	}
}

inline void StreamReader::read(LengthDelimited& t)
{
	VarInt len;
	read(len);
	t.data.resize(len.v); // TODO: not exceptionally safe
	for (size_t s = 0; s < len.v; s++) {
		// TODO: optimize
		t.data[s] = readb();
	}
}

inline void StreamReader::read(Fixed64& t)
{
	uint32_t p1 = 0;
	uint32_t p2 = 0;
	p1 = (static_cast<uint32_t>(readb())) |
		 (static_cast<uint32_t>(readb()) << 8) |
		 (static_cast<uint32_t>(readb()) << 16) |
		 (static_cast<uint32_t>(readb()) << 24);
	p2 = (static_cast<uint32_t>(readb())) |
		 (static_cast<uint32_t>(readb()) << 8) |
		 (static_cast<uint32_t>(readb()) << 16) |
		 (static_cast<uint32_t>(readb()) << 24);
	t.v = static_cast<uint64_t>(p1) | static_cast<uint64_t>(p2);
}

`)
}
