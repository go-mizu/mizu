import { Container, Title, Text, Paper, Group, Stack, Button, Grid, Badge } from '@mantine/core'
import { IconHeart, IconFlame, IconBolt, IconCrown, IconShield } from '@tabler/icons-react'
import { useAuthStore } from '../stores/auth'
import { notifications } from '@mantine/notifications'

interface ShopItem {
  id: string
  name: string
  description: string
  icon: React.ReactNode
  price: number
  color: string
}

const shopItems: ShopItem[] = [
  {
    id: 'heart_refill',
    name: 'Heart Refill',
    description: 'Refill all 5 hearts instantly',
    icon: <IconHeart size={32} />,
    price: 350,
    color: '#ff4b4b',
  },
  {
    id: 'streak_freeze',
    name: 'Streak Freeze',
    description: 'Protect your streak for one day',
    icon: <IconShield size={32} />,
    price: 200,
    color: '#1cb0f6',
  },
  {
    id: 'xp_boost',
    name: 'XP Boost',
    description: 'Earn double XP for 15 minutes',
    icon: <IconBolt size={32} />,
    price: 100,
    color: '#ffc800',
  },
  {
    id: 'double_or_nothing',
    name: 'Double or Nothing',
    description: 'Risk your streak for double gems',
    icon: <IconFlame size={32} />,
    price: 50,
    color: '#ff9600',
  },
]

export default function Shop() {
  const { user, updateUser } = useAuthStore()

  const handlePurchase = (item: ShopItem) => {
    if ((user?.gems || 0) < item.price) {
      notifications.show({
        title: 'Not enough gems',
        message: `You need ${item.price - (user?.gems || 0)} more gems`,
        color: 'red',
      })
      return
    }

    // Deduct gems
    updateUser({ gems: (user?.gems || 0) - item.price })

    // Apply item effect
    if (item.id === 'heart_refill') {
      updateUser({ hearts: 5 })
    }

    notifications.show({
      title: 'Purchase successful!',
      message: `You bought ${item.name}`,
      color: 'green',
    })
  }

  return (
    <Container size="md">
      {/* Gems Display */}
      <Paper
        p="xl"
        radius="lg"
        mb="xl"
        style={{
          backgroundColor: '#1a2c33',
          textAlign: 'center',
        }}
      >
        <Text size="lg" fw={600} style={{ color: '#8fa8b2' }} mb="xs">
          Your Gems
        </Text>
        <Group justify="center" gap="xs">
          <Text size="3rem">💎</Text>
          <Title style={{ color: '#1cb0f6', fontSize: 48 }}>{user?.gems || 0}</Title>
        </Group>
      </Paper>

      {/* Shop Items */}
      <Title order={3} mb="lg" style={{ color: 'white' }}>
        Power-ups
      </Title>
      <Grid mb="xl">
        {shopItems.map((item) => (
          <Grid.Col key={item.id} span={6}>
            <Paper
              p="xl"
              radius="lg"
              style={{
                backgroundColor: '#1a2c33',
                height: '100%',
              }}
            >
              <Stack align="center" gap="md">
                <div
                  style={{
                    width: 70,
                    height: 70,
                    borderRadius: '50%',
                    backgroundColor: `${item.color}20`,
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    color: item.color,
                  }}
                >
                  {item.icon}
                </div>
                <div style={{ textAlign: 'center' }}>
                  <Text fw={700} style={{ color: 'white' }}>{item.name}</Text>
                  <Text size="sm" style={{ color: '#8fa8b2' }}>{item.description}</Text>
                </div>
                <Button
                  fullWidth
                  variant="filled"
                  color="blue"
                  onClick={() => handlePurchase(item)}
                  disabled={(user?.gems || 0) < item.price}
                >
                  <Group gap="xs">
                    <Text>💎</Text>
                    <Text fw={700}>{item.price}</Text>
                  </Group>
                </Button>
              </Stack>
            </Paper>
          </Grid.Col>
        ))}
      </Grid>

      {/* Super Lingo */}
      <Title order={3} mb="lg" style={{ color: 'white' }}>
        Super Lingo
      </Title>
      <Paper
        p="xl"
        radius="lg"
        style={{
          background: 'linear-gradient(135deg, #ff9600 0%, #ffc800 100%)',
        }}
      >
        <Group justify="space-between" align="center">
          <div>
            <Group gap="xs" mb="xs">
              <IconCrown size={28} style={{ color: 'white' }} />
              <Title order={2} style={{ color: 'white' }}>Super Lingo</Title>
            </Group>
            <Stack gap="xs">
              <Group gap="xs">
                <IconHeart size={18} style={{ color: 'white' }} />
                <Text style={{ color: 'white' }}>Unlimited Hearts</Text>
              </Group>
              <Group gap="xs">
                <IconShield size={18} style={{ color: 'white' }} />
                <Text style={{ color: 'white' }}>Unlimited Streak Freezes</Text>
              </Group>
              <Group gap="xs">
                <IconBolt size={18} style={{ color: 'white' }} />
                <Text style={{ color: 'white' }}>No Ads</Text>
              </Group>
              <Group gap="xs">
                <IconCrown size={18} style={{ color: 'white' }} />
                <Text style={{ color: 'white' }}>Practice Hub Access</Text>
              </Group>
            </Stack>
          </div>
          <Stack align="center">
            <Badge size="xl" color="dark" style={{ backgroundColor: 'rgba(0,0,0,0.3)' }}>
              FREE TRIAL
            </Badge>
            <Button size="lg" color="dark" radius="xl">
              Try 2 Weeks Free
            </Button>
          </Stack>
        </Group>
      </Paper>

      {/* Earn More Gems */}
      <Paper
        p="xl"
        radius="lg"
        mt="xl"
        style={{
          backgroundColor: '#1a2c33',
        }}
      >
        <Title order={4} mb="md" style={{ color: 'white' }}>
          Earn More Gems
        </Title>
        <Stack gap="md">
          <Group justify="space-between">
            <Text style={{ color: '#8fa8b2' }}>Complete a lesson</Text>
            <Badge color="blue">+5-15 💎</Badge>
          </Group>
          <Group justify="space-between">
            <Text style={{ color: '#8fa8b2' }}>Perfect lesson (no mistakes)</Text>
            <Badge color="blue">+5 💎 bonus</Badge>
          </Group>
          <Group justify="space-between">
            <Text style={{ color: '#8fa8b2' }}>Complete friend quest</Text>
            <Badge color="blue">+100 💎</Badge>
          </Group>
          <Group justify="space-between">
            <Text style={{ color: '#8fa8b2' }}>Win league</Text>
            <Badge color="blue">+50-100 💎</Badge>
          </Group>
        </Stack>
      </Paper>
    </Container>
  )
}
